package transport

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	upgradeFailed = "Upgrade failed: "

	WsDefaultPingInterval   = 30 * time.Second
	WsDefaultPingTimeout    = 30 * time.Second
	WsDefaultReceiveTimeout = 30 * time.Second
	WsDefaultSendTimeout    = 30 * time.Second
	WsDefaultBufferSize     = 1024 * 32
)

var (
	ErrorBinaryMessage     = errors.New("Binary messages are not supported")
	ErrorBadBuffer         = errors.New("Buffer error")
	ErrorPacketWrong       = errors.New("Wrong packet type error")
	ErrorMethodNotAllowed  = errors.New("Method not allowed")
	ErrorHttpUpgradeFailed = errors.New("Http upgrade failed")
)

type WebsocketConnection struct {
	socket    *websocket.Conn
	transport *WebsocketTransport
}

func (wsc *WebsocketConnection) GetMessage() (message string, err error) {
	if wsc.transport.Status == StatusReconnecting {
		return "", nil
	}

	wsc.socket.SetReadDeadline(time.Now().Add(wsc.transport.ReceiveTimeout))
	msgType, reader, err := wsc.socket.NextReader()
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			log.Println("Connection closed normally:", err)
			// TODO: close the client
			wsc.transport.Status = StatusClosed
			return "", err
		} else {
			log.Printf("Unexpected error: %v, attempting to reconnect...", err)
			if !wsc.transport.SendReconnectSignal() {
				log.Println("reconnect failed, close the client")
				// TODO: close the client
				return "", err
			} else {
				return "", nil
			}
		}
	}

	//support only text messages exchange
	if msgType != websocket.TextMessage {
		return "", ErrorBinaryMessage
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", ErrorBadBuffer
	}
	text := string(data)

	//empty messages are not allowed
	if len(text) == 0 {
		return "", ErrorPacketWrong
	}
	return text, nil
}

func (wsc *WebsocketConnection) WriteMessage(message string) error {
	if wsc.transport.Status == StatusReconnecting {
		return nil
	}
	wsc.socket.SetWriteDeadline(time.Now().Add(wsc.transport.SendTimeout))
	writer, err := wsc.socket.NextWriter(websocket.TextMessage)
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			log.Println("Connection closed normally:", err)
			// TODO: close the client
			wsc.transport.Status = StatusClosed
			return err
		} else {
			log.Printf("Unexpected error: %v, attempting to reconnect...", err)
			if !wsc.transport.SendReconnectSignal() {
				log.Println("reconnect failed, close the client")
				// TODO: close the client
				return err
			} else {
				return nil
			}
		}
	}

	if _, err := writer.Write([]byte(message)); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return nil
}

func (wsc *WebsocketConnection) Close() {
	err := wsc.socket.Close()
	if err != nil {
		log.Println("Failed to close websocket connection:", err)
	}
}

func (wsc *WebsocketConnection) PingParams() (interval, timeout time.Duration) {
	return wsc.transport.PingInterval, wsc.transport.PingTimeout
}

const (
	StatusConnected = iota
	StatusDisconnected
	StatusReconnecting
	StatusClosed
)

type WebsocketTransport struct {
	PingInterval   time.Duration
	PingTimeout    time.Duration
	ReceiveTimeout time.Duration
	SendTimeout    time.Duration

	BufferSize  int
	UnsecureTLS bool

	RequestHeader  http.Header
	AutoReconnect  bool
	ReconnectDelay time.Duration
	ReconnectMax   int

	// reconnect related
	ReConnectChan  chan struct{}
	ReconnectCount int
	Status         int
}

func (wst *WebsocketTransport) Handshake(url string, namespace string) (sid string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	// read the response body and parse to string
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// remove the first '0' character
	bodyStr := string(body)[1:]
	// parse the string to json
	if err := json.Unmarshal([]byte(bodyStr), &result); err != nil {
		return "", err
	}

	sid, ok := result["sid"].(string)
	if !ok {
		return "", errors.New("failed to get sid from handshake response")
	}

	// second handshake: send the namespace parameter
	req, err := http.NewRequest("POST", url+"&sid="+sid, strings.NewReader("40/"+namespace+","))
	if err != nil {
		return "", err
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// third req：send the websocket upgrade request
	req, err = http.NewRequest("GET", url+"&sid="+sid, nil)
	if err != nil {
		return "", err
	}
	//FIXME: cost time too much!
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New("failed to upgrade to websocket")
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	bodyStr = string(body)
	// case it can return 40 represent success or 42 represent message coming
	if strings.Index(bodyStr, "40") != 0 && strings.Index(bodyStr, "42") != 0 {
		return "", errors.New("failed to handshake, check your auth or namespace:" + bodyStr)
	}
	return sid, nil
}

func (wst *WebsocketTransport) Connect(url string) (conn Connection, err error) {
	dialer := websocket.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: wst.UnsecureTLS}}
	socket, _, err := dialer.Dial(url, wst.RequestHeader)
	if err != nil {
		return nil, err
	}
	wst.Status = StatusConnected
	return &WebsocketConnection{socket, wst}, nil
}

func (wst *WebsocketTransport) HandleConnection(
	w http.ResponseWriter, r *http.Request) (conn Connection, err error) {

	if r.Method != "GET" {
		http.Error(w, upgradeFailed+ErrorMethodNotAllowed.Error(), 503)
		return nil, ErrorMethodNotAllowed
	}

	socket, err := websocket.Upgrade(w, r, nil, wst.BufferSize, wst.BufferSize)
	if err != nil {
		http.Error(w, upgradeFailed+err.Error(), 503)
		return nil, ErrorHttpUpgradeFailed
	}

	return &WebsocketConnection{socket, wst}, nil
}

/*
*
Websocket connection do not require any additional processing
*/
func (wst *WebsocketTransport) Serve(w http.ResponseWriter, r *http.Request) {}

/*
*
Returns websocket connection with default params
*/
func GetDefaultWebsocketTransport() *WebsocketTransport {
	return &WebsocketTransport{
		PingInterval:   WsDefaultPingInterval,
		PingTimeout:    WsDefaultPingTimeout,
		ReceiveTimeout: WsDefaultReceiveTimeout,
		SendTimeout:    WsDefaultSendTimeout,
		BufferSize:     WsDefaultBufferSize,
		UnsecureTLS:    false,
		ReconnectDelay: 5 * time.Second,
		ReconnectMax:   10,
		ReConnectChan:  make(chan struct{}),
	}
}

func (wst *WebsocketTransport) SetUnsecureTLS(unsecureTLS bool) {
	wst.UnsecureTLS = unsecureTLS
}

func (wst *WebsocketTransport) SendReconnectSignal() bool {
	// receive the reconnect signal, and lost the message
	if wst.Status == StatusConnected || wst.Status == StatusDisconnected {
		wst.Status = StatusReconnecting
		wst.ReConnectChan <- struct{}{}
	} else if wst.Status == StatusReconnecting {
		// TODO: handle the message lost
		log.Println("reconnecting, ignore this message")
	} else {
		return false
	}
	return true
}

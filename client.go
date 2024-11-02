package socketioclient

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/liangsqrt/socketio-client-go/transport"
)

const (
	webSocketProtocol       = "ws://"
	webSocketSecureProtocol = "wss://"
	socketioUrl             = "/socket.io/?EIO=4&transport=websocket"
)

/*
*
Socket.io client representation
*/
type Client struct {
	*Namespace
	methods
	Channel
	Conf ConConf
	Tr   transport.Transport
}

type ConConf struct {
	Host   string
	Port   int
	Secure bool
	Query  map[string]string
}

/*
*
Generate websocket URL from ConConf
*/
func (s *ConConf) GenerateWebSocketUrl(sid string) string {
	var protocol string
	if s.Secure {
		protocol = webSocketSecureProtocol
	} else {
		protocol = webSocketProtocol
	}

	queryParams := url.Values{}
	for key, value := range s.Query {
		queryParams.Add(key, value)
	}
	queryParams.Add("sid", sid)
	//queryParams.Add("EIO", "4")
	//queryParams.Add("transport", "websocket")

	return protocol + net.JoinHostPort(s.Host, strconv.Itoa(s.Port)) + socketioUrl + "&" + queryParams.Encode()
}

/*
*
Generate handshake URL from ConConf
*/
func (s *ConConf) GenerateHandshakeUrl() string {
	var protocol string
	if s.Secure {
		protocol = "https://"
	} else {
		protocol = "http://"
	}

	queryParams := url.Values{}
	for key, value := range s.Query {
		queryParams.Add(key, value)
	}
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	queryParams.Add("t", timestamp)
	return protocol + net.JoinHostPort(s.Host, strconv.Itoa(s.Port)) + "/socket.io/?EIO=4&transport=polling" + "&" + queryParams.Encode()
}

/*
*
Get ws/wss url by host and port
*/
func GetUrl(socketUrl ConConf) string {

	var prefix string
	if socketUrl.Secure {
		prefix = webSocketSecureProtocol
	} else {
		prefix = webSocketProtocol
	}
	return prefix + net.JoinHostPort(socketUrl.Host, strconv.Itoa(socketUrl.Port)) + socketioUrl
}

func (c *Client) initNamespace(sid string) error {
	c.Namespace.Sid = sid
	//msg := protocol.Message{
	//	EngineIoType: protocol.EngineMessageTypeMessage,
	//	SocketType:   protocol.SocketMessageTypeConnect,
	//	SocketEvent: protocol.SocketEvent{
	//		NS: c.Namespace.Namespace,
	//	},
	//}
	//
	//command, err := protocol.Encode(&msg)
	//if err != nil {
	//	return err
	//}
	//
	//err = send(command, &c.Channel)
	//if err != nil {
	//	return err
	//}

	c.Namespace.SetInit(true)
	return nil
}

func (c *Client) handshake() error {
	url := c.Conf.GenerateHandshakeUrl()
	// frist req：get the sid
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	// read the response body and parse to string
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// remove the first '0' character
	bodyStr := string(body)[1:]
	// parse the string to json
	if err := json.Unmarshal([]byte(bodyStr), &result); err != nil {
		return err
	}

	sid, ok := result["sid"].(string)
	if !ok {
		return errors.New("failed to get sid from handshake response")
	}

	// second handshake: send the namespace parameter
	req, err := http.NewRequest("POST", url+"&sid="+sid+"&EIO=4&transport=polling", strings.NewReader("40/"+c.Namespace.Namespace+","))
	if err != nil {
		return err
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	// body, err = io.ReadAll(resp.Body)
	// if err != nil {
	// 	return err
	// }
	// if err := json.Unmarshal([]byte(body), &result); err != nil {
	// 	return err
	// }
	// print(result)
	defer resp.Body.Close()

	// third req：send the websocket upgrade request
	req, err = http.NewRequest("GET", url+"&sid="+sid, nil)
	if err != nil {
		return err
	}
	//FIXME: cost time too much!
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("failed to upgrade to websocket")
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	bodyStr = string(body)
	if strings.Index(bodyStr, "40") != 0 {
		return errors.New("failed to handshake, check your auth or namespace")
	}
	err = c.initNamespace(sid)
	if err != nil {
		return err
	}
	return nil
}

// Dial
// The correct ws protocol url example:
// ws://localhost:8080/socket.io/?EIO=3&transport=websocket&sid=xxxx
// /*
func Dial(conConf ConConf, tr transport.Transport, ns *Namespace) (*Client, error) {
	c := &Client{Conf: conConf, Tr: tr}
	err := c.HandShake(ns)
	if err != nil {
		return nil, err
	}
	err = c.Connect()
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			cnt := <-ReconnectChan
			if cnt > 0 {
				err := c.Connect()
				if err == nil {
					ReconnectCntLock.Lock()
					ReconnectCnt = 10
					ReconnectCntLock.Unlock()
				}
			}
		}
	}()
	return c, nil
}

func (c *Client) HandShake(ns *Namespace) error {
	c.Namespace = ns
	c.initChannel()
	c.initMethods()
	if err := c.handshake(); err != nil {
		log.Fatalln("handshake failed", err)
		return err
	}
	return nil
}

func (c *Client) Connect() error {
	wsUrl := c.Conf.GenerateWebSocketUrl(c.Namespace.Sid)
	var err error
	c.conn, err = c.Tr.Connect(wsUrl)
	if err != nil {
		return err
	}
	err = handleUpgrade(&c.Channel, &c.methods)
	if err != nil {
		return err
	}

	go inLoop(&c.Channel, &c.methods)
	go outLoop(&c.Channel, &c.methods)
	go pinger(&c.Channel)
	return nil
}

/*
*
Close client connection
*/
func (c *Client) Close() {
	err := closeChannel(&c.Channel, &c.methods)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (c *Client) Emit(method string, args interface{}) error {
	return c.Channel.emit(method, c.Namespace.Namespace, args)
}

func (c *Client) On(method string, callback interface{}) {
	c.methods.On(method, callback)
}

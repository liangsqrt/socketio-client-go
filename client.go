package socketioclient

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
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
}

type SocketIoUrl struct {
	Host   string
	Port   int
	Secure bool
	Query  map[string]string
}

/*
*
Generate websocket URL from SocketIoUrl
*/
func (s *SocketIoUrl) GenerateWebSocketUrl(sid string) string {
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
Generate handshake URL from SocketIoUrl
*/
func (s *SocketIoUrl) GenerateHandshakeUrl() string {
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
func GetUrl(socketUrl SocketIoUrl) string {

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

func (c *Client) handshake(socketUrl SocketIoUrl) error {
	url := socketUrl.GenerateHandshakeUrl()
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
		return errors.New("failed to handshake, check your auth info")
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
func Dial(url SocketIoUrl, tr transport.Transport, ns *Namespace, gracefulExit bool) (*Client, error) {
	c := &Client{}
	c.Namespace = ns
	c.initChannel()
	c.initMethods()
	if err := c.handshake(url); err != nil {
		log.Fatalln("handshake failed", err)
		return nil, err
	}

	var err error
	wsUrl := url.GenerateWebSocketUrl(c.Namespace.Sid)
	c.conn, err = tr.Connect(wsUrl)
	if err != nil {
		return nil, err
	}
	handleUpgrade(&c.Channel, &c.methods)

	go inLoop(&c.Channel, &c.methods)
	go outLoop(&c.Channel, &c.methods)
	go pinger(&c.Channel)

	if gracefulExit {
		// graceful exit method
		cleanup := func() {
			err := c.Emit("disconnect", nil)
			if err != nil {
				log.Println("Failed to send disconnect signal:", err)
			}
			c.Close()
		}

		// catch program exit signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			cleanup()
			os.Exit(0)
		}()
	}

	return c, nil
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

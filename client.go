package socketioclient

import (
	"errors"
	"log"
	"net"
	"net/url"
	"strconv"
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
	Conf            ConConf
	ReconnectCount  int
	LastConnectTime time.Time
	ReconnectDelay  time.Duration
	ReconnectMax    int
	Tr              transport.Transport
}

type ConConf struct {
	Host           string
	Port           int
	Secure         bool
	Query          map[string]string
	AutoReconnect  bool
	ReconnectDelay time.Duration
	ReconnectMax   int
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
	// c.Namespace.SetInit(true)
	return nil
}

func (c *Client) handshake() error {
	url := c.Conf.GenerateHandshakeUrl()
	// frist req：get the sid
	sid, err := c.Tr.Handshake(url, c.Namespace.Namespace)
	if err != nil {
		return err
	}
	c.initNamespace(sid)
	return nil
}

// Dial
// The correct ws protocol url example:
// ws://localhost:8080/socket.io/?EIO=3&transport=websocket&sid=xxxx
// /*
func Dial(conConf ConConf, tr transport.Transport, ns *Namespace) (*Client, error) {
	c := &Client{Conf: conConf, Tr: tr, Namespace: ns}
	c.Namespace = ns
	c.initChannel()
	c.initMethods()
	err := c.Connect()
	if err != nil {
		return nil, err
	}

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
	handShakeUrl := c.Conf.GenerateHandshakeUrl()
	var err error
	if sid, err := c.Tr.Handshake(handShakeUrl, c.Namespace.Namespace); err != nil {
		return err
	} else {
		c.initNamespace(sid)
	}
	wsUrl := c.Conf.GenerateWebSocketUrl(c.Namespace.Sid)
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
	go func() {
		for range c.Tr.(*transport.WebsocketTransport).ReConnectChan {
			if conn, err := c.Reconnect(); err != nil {
				log.Println("reconnect failed", err)
			} else {
				log.Println("reconnect success")
				c.conn = conn
			}
		}
	}()
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

func (c *Client) Reconnect() (conn transport.Connection, err error) {
	for c.ReconnectCount < c.ReconnectMax {
		c.ReconnectCount++
		// 指数退避
		delay := c.ReconnectDelay * time.Duration(c.ReconnectCount)
		if time.Since(c.LastConnectTime) < delay {
			time.Sleep(delay - time.Since(c.LastConnectTime))
		}
		if sid, err := c.Tr.Handshake(c.Conf.GenerateHandshakeUrl(), c.Namespace.Namespace); err != nil {
			log.Println("reconnect failed", err)
		} else {
			c.initNamespace(sid)
		}
		if conn, err := c.Tr.Connect(c.Conf.GenerateWebSocketUrl(c.Namespace.Sid)); err != nil {
			log.Println("reconnect failed", err)
		} else {
			c.ReconnectCount = 0
			return conn, nil
		}
	}
	return nil, errors.New("reconnect failed")
}

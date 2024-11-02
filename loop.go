package socketioclient

import (
	"encoding/json"
	"errors"
	"github.com/liangsqrt/socketio-client-go/protocol"
	"github.com/liangsqrt/socketio-client-go/transport"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	queueBufferSize = 10000
)

var (
	ErrorWrongHeader = errors.New("Wrong header")
	ReconnectChan    = make(chan int, 10)
	ReconnectCnt     = 10
	ReconnectCntLock = sync.Mutex{}
)

/*
*
engine.io header to send or receive
*/
type Header struct {
	Sid          string `json:"sid"`
	PingInterval int    `json:"pingInterval"`
	PingTimeout  int    `json:"pingTimeout"`
	MaxPayload   int    `json:"maxPayload"`
}

/*
*
socket.io connection handler

use IsAlive to check that handler is still working
use Dial to connect to websocket
use In and Out channels for message exchange
Close message means channel is closed
ping is automatic
*/
type Channel struct {
	conn transport.Connection

	out    chan string
	header Header

	alive     bool
	aliveLock sync.Mutex

	//ack ackProcessor

	//server  *Server
	ip      string
	request *http.Request
}

/*
*
create channel, map, and set active
*/
func (c *Channel) initChannel() {
	//TODO: queueBufferSize from constant to server or client variable
	c.out = make(chan string, queueBufferSize)
	//c.ack.resultWaiters = make(map[int](chan string))
	c.setAliveValue(true)
}

/*
*
Get id of current socket connection
*/
func (c *Channel) Id() string {
	return c.header.Sid
}

func (c *Channel) SetHeader(pkg string) {
	err := json.Unmarshal([]byte(pkg), &c.header)
	if err != nil {
		log.Println("ERROR decoding header: " + err.Error())
	}

}

/*
*
Checks that Channel is still alive
*/
func (c *Channel) IsAlive() bool {
	c.aliveLock.Lock()
	isAlive := c.alive
	c.aliveLock.Unlock()

	return isAlive
}

func (c *Channel) setAliveValue(value bool) {
	c.aliveLock.Lock()
	c.alive = value
	c.aliveLock.Unlock()
}

/*
*
Close channel
*/
func closeChannel(c *Channel, m *methods, args ...interface{}) error {
	log.Println("Channel closed - calling disconnect")
	c.conn.Close()
	c.setAliveValue(false)

	//clean outloop
	for len(c.out) > 0 {
		<-c.out
	}

	deleteOverflooded(c)
	f, _ := m.findMethod("disconnection")
	m.initMethods()
	if f != nil {
		f.callFunc(c, &struct{}{})
	}

	return nil
}

// incoming messages loop, puts incoming messages to In channel
func inLoop(c *Channel, m *methods) error {
	for {
		pkg, err := c.conn.GetMessage()
		if err != nil {
			ReconnectCntLock.Lock()
			if ReconnectCnt > 0 {
				ReconnectCnt -= 1
				ReconnectChan <- ReconnectCnt
			} else {
				return closeChannel(c, m, err)
			}
			ReconnectCntLock.Unlock()
			return nil
		}
		engineIoType, err := protocol.GetEngineMessageType(pkg)
		//logger.LogDebugSocketIo("Engine IO type: " + engineIoType.String())
		if err != nil {
			err := closeChannel(c, m, protocol.ErrorWrongPacket)
			if err != nil {
				return err
			}
			return err
		}
		go m.processIncomingMessage(c, engineIoType, pkg)
	}

}

var overflooded sync.Map

func deleteOverflooded(c *Channel) {
	overflooded.Delete(c)
}

func storeOverflow(c *Channel) {
	overflooded.Store(c, struct{}{})
}

/*
*
outgoing messages loop, sends messages from channel to socket
*/
func outLoop(c *Channel, m *methods) error {
	for {
		outBufferLen := len(c.out)
		if outBufferLen >= queueBufferSize-1 {
			return closeChannel(c, m, ErrorSocketOverflood)
		} else if outBufferLen > int(queueBufferSize/2) {
			storeOverflow(c)
		} else {
			deleteOverflooded(c)
		}

		msg := <-c.out

		err := c.conn.WriteMessage(msg)
		if err != nil {
			ReconnectCntLock.Lock()
			if ReconnectCnt > 0 {
				ReconnectCnt -= 1
				ReconnectChan <- ReconnectCnt
			} else {
				return closeChannel(c, m, err)
			}
			ReconnectCntLock.Unlock()
		}

	}
}

/*
*
Pinger sends ping messages for keeping connection alive
*/
func pinger(c *Channel) {
	interval, _ := c.conn.PingParams()
	ticker := time.NewTicker(interval)

	for {
		<-ticker.C
		if !c.IsAlive() {
			return
		}
		c.out <- protocol.PingMessage
	}
}

/*
*
Handle the upgrade process
*/
func handleUpgrade(c *Channel, m *methods) error {
	// Step 1: Send 2probe message

	msgList := []string{"2probe", "5"}
	for _, msg := range msgList {
		err := c.conn.WriteMessage(msg)
		if err != nil {
			return err
		}
		// Step 2: Wait for 3probe response
		ansMsg, err := c.conn.GetMessage()
		if err != nil || !(ansMsg == "3probe" || ansMsg == "6" || ansMsg == "2" || strings.HasPrefix(ansMsg, "40")) {
			return closeChannel(c, m, errors.New("failed in handshake message"))
		}

	}
	return nil

}

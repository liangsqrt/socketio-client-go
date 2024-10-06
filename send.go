package socketioclient

import (
	"errors"
	"log"
	"socketio-client-go/protocol"
)

var (
	ErrorSendTimeout     = errors.New("Timeout")
	ErrorSocketOverflood = errors.New("Socket overflood")
)

/*
*
Send message packet to socket
*/
func send(msg string, c *Channel) error {
	//preventing json/encoding "index out of range" panic
	defer func() {
		if r := recover(); r != nil {
			log.Println("socket.io send panic: ", r)
		}
	}()

	if len(c.out) == queueBufferSize {
		return ErrorSocketOverflood
	}

	c.out <- msg

	return nil
}

/*
*
Create packet based on given data and send it
*/
func (c *Channel) emit(method string, namespace string, args interface{}) error {

	msg := protocol.Message{
		EngineIoType: protocol.EngineMessageTypeMessage,
		SocketType:   protocol.SocketMessageTypeEvent,
	}
	//content, _ := json.Marshal(args)
	msg.SocketEvent.EventName = method
	msg.SocketEvent.NS = namespace
	msg.SocketEvent.EventContent = args

	command, err := protocol.Encode(&msg)
	if err != nil {
		return err
	}

	err = send(command, c)
	if err != nil {
		return err
	}
	return nil
}

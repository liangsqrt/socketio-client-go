package protocol

/*
https://github.com/socketio/socket.io-protocol#0---connect
*/
type EngineMessageType int64

const (
	EngineMessageTypeOpen    EngineMessageType = 0
	EngineMessageTypeClose   EngineMessageType = 1
	EngineMessageTypePing    EngineMessageType = 2
	EngineMessageTypePong    EngineMessageType = 3
	EngineMessageTypeMessage EngineMessageType = 4
	EngineMessageTypeUpgrade EngineMessageType = 5
	EngineMessageTypeNoop    EngineMessageType = 6
)

/*https://github.com/socketio/socket.io-protocol#examples*/

type SocketMessageType int64

const (
	SocketMessageTypeConnect      SocketMessageType = 0
	SocketMessageTypeDisconnect   SocketMessageType = 1
	SocketMessageTypeEvent        SocketMessageType = 2
	SocketMessageTypeAck          SocketMessageType = 3
	SocketMessageTypeConnectError SocketMessageType = 4
	SocketMessageTypeBinaryEvent  SocketMessageType = 5
	SocketMessageTypeBinaryAck    SocketMessageType = 6
	SocketMessageTypeNone         SocketMessageType = 99
)

func (e EngineMessageType) String() string {
	switch e {
	case
		EngineMessageTypeOpen:
		return "EngineMessageTypeOpen"
	case
		EngineMessageTypeClose:
		return "EngineMessageTypeClose"
	case
		EngineMessageTypePing:
		return "EngineMessageTypePing"
	case
		EngineMessageTypePong:
		return "EngineMessageTypePong"
	case
		EngineMessageTypeMessage:
		return "EngineMessageTypeMessage"
	case
		EngineMessageTypeUpgrade:
		return "EngineMessageTypeUpgrade"
	case
		EngineMessageTypeNoop:
		return "EngineMessageTypeNoop"
	}
	return "Unknown"
}

func (e SocketMessageType) String() string {
	switch e {

	case SocketMessageTypeConnect:
		return "SocketMessageTypeConnect"
	case SocketMessageTypeDisconnect:
		return "SocketMessageTypeDisconnect"
	case SocketMessageTypeEvent:
		return "SocketMessageTypeEvent"
	case SocketMessageTypeAck:
		return "SocketMessageTypeAck"
	case SocketMessageTypeConnectError:
		return "SocketMessageTypeConnectError"
	case SocketMessageTypeBinaryEvent:
		return "SocketMessageTypeBinaryEvent"
	case SocketMessageTypeBinaryAck:
		return "SocketMessageTypeBinaryAck"
	case SocketMessageTypeNone:
		return "SocketMessageTypeNone"
	}
	return "Unknown"
}

type Message struct {
	EngineIoType EngineMessageType
	SocketType   SocketMessageType
	SocketEvent  SocketEvent
}
type SocketEventArray struct {
	Emit []string
}
type SocketEvent struct {
	NS           string
	EventName    string
	EventContent interface{}
}

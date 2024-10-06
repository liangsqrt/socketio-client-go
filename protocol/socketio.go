package protocol

import (
	"encoding/json"
	"errors"
	"log"
	"regexp"
	"strconv"
)

const (
	open          = "0"
	msg           = "4"
	emptyMessage  = "40"
	commonMessage = "42"
	svrAnsMessage = "43"

	CloseMessage = "1"
	PingMessage  = "3" // the ping in client is 3 rather than 2
	PongMessage  = "2"
)

var (
	ErrorWrongMessageType = errors.New("wrong message type")
	ErrorWrongPacket      = errors.New("wrong packet")
)

func Encode(msg *Message) (string, error) {
	output := ""
	output += strconv.Itoa(int(msg.EngineIoType))

	if msg.SocketType == SocketMessageTypeNone {
		return output, nil
	}
	output += strconv.Itoa(int(msg.SocketType))
	output += "/" + msg.SocketEvent.NS + ",0"
	if !(msg.SocketEvent.EventName == "" || msg.SocketEvent.EventContent == "") {
		payload, _ := json.Marshal([2]interface{}{msg.SocketEvent.EventName, msg.SocketEvent.EventContent})
		output += string(payload)
	}

	return output, nil
}

func GetEngineMessageType(data string) (EngineMessageType, error) {
	if len(data) == 0 {
		return 0, ErrorWrongMessageType
	}
	msgType, _ := strconv.Atoi(data[0:1])
	if msgType > 6 {
		return 0, ErrorWrongMessageType
	}
	return EngineMessageType(msgType), nil
}

func GetSocketMessageType(data string) (SocketMessageType, error) {
	if len(data) == 0 {
		return 0, ErrorWrongMessageType
	}
	msgType, _ := strconv.Atoi(data[1:2])
	if msgType > 6 {
		return 0, ErrorWrongMessageType
	}
	return SocketMessageType(msgType), nil
}

func GetSocketIoMessage(data string) (*Message, error) {
	re := regexp.MustCompile(`(\d{1,2})/([^,]*),(.+)`)
	matches := re.FindStringSubmatch(data)
	if len(matches) == 4 {
		// if len(matches[1]) == 2 {
		// 	engineIoType, _ := strconv.Atoi(string(matches[1][0]))
		// 	msgType, _ := strconv.Atoi(string(matches[1][1]))

		// } else {
		// 	engineIoType, _ := strconv.Atoi(string(matches[1]))
		// 	engineIoType = engineIoType.(EngineMessageType)
		// 	// msgType, _ := strconv.Atoi(string(matches[2]))
		// 	msgType := SocketMessageTypeNone
		// }
		namespace := matches[2]
		jsonData := matches[3]

		// 解析JSON数据
		var result []interface{}
		if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
			log.Fatalln("JSON parse error:", err)
			return nil, err
		}

		eventName := result[0].(string)
		var payload interface{}
		switch result[1].(type) {
		case string:
			payload = result[1].(string)
		case map[string]interface{}:
			payload = result[1].(map[string]interface{})
		default:
			log.Fatalln("unknown type")
			return nil, errors.New("unknown type")
		}

		return &Message{
			EngineIoType: EngineMessageTypeMessage,
			SocketType:   SocketMessageTypeEvent,
			SocketEvent: SocketEvent{
				EventName:    eventName,
				EventContent: payload,
				NS:           namespace,
			},
		}, nil
	} else {
		return nil, errors.New("data format not match")
	}

}

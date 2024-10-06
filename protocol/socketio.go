package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"socketio-client-go/logger"
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

func GetSocketIoEmitName(data string) string {
	re := regexp.MustCompile(`\d{1,2}(/[^,]*),(.+)`)
	matches := re.FindStringSubmatch(data)
	if len(matches) == 4 {
		namespace := matches[1]
		jsonData := matches[2]

		// 解析JSON数据
		var result []interface{}
		if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
			fmt.Println("JSON解析错误:", err)
			return data
		}

		eventName := result[0].(string)
		payload := result[1].(map[string]interface{})

		fmt.Println("Namespace:", namespace)
		fmt.Println("Event Name:", eventName)
		fmt.Println("Payload:", payload)
	} else {
		fmt.Println("数据格式不匹配")
	}

	jsonevent := data[2:]
	var emit []interface{}
	err := json.Unmarshal([]byte(jsonevent), &emit)
	if err != nil {
		logger.LogErrorSocketIo("Error: " + err.Error())
	}
	emitNameString, _ := emit[0].(string)
	return emitNameString
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
			fmt.Println("JSON parse error:", err)
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
			fmt.Println("unknown type")
			return nil, errors.New("unknown type")
		}

		fmt.Println("Namespace:", namespace)
		fmt.Println("Event Name:", eventName)
		fmt.Println("Payload:", payload)
		return &Message{
			EngineIoType: EngineMessageTypeMessage,
			SocketType:   SocketMessageTypeEvent,
			SocketEvent: SocketEvent{
				EventName:    eventName,
				EventContent: payload,
			},
		}, nil
	} else {
		return nil, errors.New("data format not match")
	}

}

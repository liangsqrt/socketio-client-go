package test

import (
	"testing"
	"time"

	socketioclient "github.com/liangsqrt/socketio-client-go"
	"github.com/liangsqrt/socketio-client-go/transport"
)

func TestClient(t *testing.T) {
	url := socketioclient.SocketIoUrl{
		Host:   "127.0.0.1",
		Port:   8080,
		Secure: false,
		Query:  map[string]string{"token": "xxxxx"},
	}
	c, err := socketioclient.Dial(
		url,
		transport.GetDefaultWebsocketTransport(),
		&socketioclient.Namespace{
			Namespace: "statistic",
		},
		true,
	)
	if err != nil {
		panic(err.Error())
	}
	c.On("node_traffic_usage", func(c *socketioclient.Channel, args []interface{}) error {
		if len(args) > 0 {
			print(args)
		} else {
			print("Unexpected argument type")
		}
		return nil
	})

	if err != nil {
		print(err.Error())
		panic(err.Error())
	}

	payload := map[string]string{
		"name":      "172.16.1.1:7890",
		"uuid":      "a1bac2b9-c320-4ea8-a7b8-7ba85a57a30d",
		"download":  "123412",
		"upload":    "12321",
		"start":     "2024-10-03 13:01:01",
		"end":       "2024-10-03 12:01:01",
		"device_id": "12",
		"node_id":   "12",
	}
	for i := 0; i < 10; i++ {
		err = c.Emit("node_traffic_usage", payload)
		time.Sleep(time.Second * 2)
	}

	if err != nil {
		print(err.Error())
	}

}

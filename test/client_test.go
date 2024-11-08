package test

import (
	"fmt"
	"testing"
	"time"

	socketioclient "github.com/liangsqrt/socketio-client-go"
	"github.com/liangsqrt/socketio-client-go/transport"
)

func TestClient(t *testing.T) {
	url := socketioclient.ConConf{
		Host:   "wsio.anydoor.world",
		Port:   80,
		Secure: false,
		Query:  map[string]string{"token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ0b2tlbl90eXBlIjoiYWNjZXNzIiwiZXhwIjoxNzMwNTgwMDIxLCJpYXQiOjE3Mjc5ODgwMjEsImp0aSI6IjU3OGE3NDg1YmRmZjRkYjliNDY1ZjgwNjA1YTAxMTM0IiwidXNlcl9pZCI6Mn0.VvQiyh4fYdic8ie5mUx48yBuI77l_y_4o1gSVOzDyO0"},
	}
	c, err := socketioclient.Dial(
		url,
		transport.GetDefaultWebsocketTransport(),
		&socketioclient.Namespace{
			Namespace: "statistic",
		},
	)
	if err != nil {
		panic(err.Error())
	}
	c.On("node_traffic_usage", func(c *socketioclient.Channel, args []interface{}) error {
		// you must make sure the callback function has the same number structure of arguments
		if len(args) > 0 {
			print(args)
		} else {
			print("Unexpected argument type")
		}
		return nil
	})
	payload := map[string]string{
		"name":      "172.16.1.1:7890",
		"uuid":      "a1bac2b9-c320-4ea8-a7b8-7ba85a57a30d",
		"download":  "123412",
		"upload":    "12321",
		"start":     "2024-10-23 13:01:01",
		"end":       "2024-10-23 12:01:01",
		"device_id": "12",
		"node_id":   "12",
	}
	for i := 0; i < 100; i++ {
		err = c.Emit("node_traffic_usage", payload)
		fmt.Printf("err is %v\n", err)
		time.Sleep(time.Second * 2)
	}

	if err != nil {
		print(err.Error())
	}

}

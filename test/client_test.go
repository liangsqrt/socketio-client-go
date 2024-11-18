package test

import (
	"testing"
	"time"

	socketioclient "github.com/liangsqrt/socketio-client-go"
	"github.com/liangsqrt/socketio-client-go/transport"
)

func TestClient(t *testing.T) {
	url := socketioclient.ConConf{
		Host:   "127.0.0.1",
		Port:   32093,
		Secure: false,
		Query:  map[string]string{"token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ0b2tlbl90eXBlIjoiYWNjZXNzIiwiZXhwIjoxNzM0NTA5NTI2LCJpYXQiOjE3MzE5MTc1MjYsImp0aSI6ImE4MTM4YjEwMDcyNjQzNDA4MjY3N2FlMWI2M2Q3MTY1IiwidXNlcl9pZCI6Mn0.YBGG702SDjAOjwKZp-20sQyFsc0ZX_5GcLfyx3xYIvo"},
	}
	defaultTransport := transport.GetDefaultWebsocketTransport()
	defaultTransport.SetUnsecureTLS(true)

	c, err := socketioclient.Dial(
		url,
		defaultTransport,
		&socketioclient.Namespace{
			Namespace: "proxy",
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
		// fmt.Printf("err is %v\n", err)
		time.Sleep(time.Second * 2)
	}

	if err != nil {
		print(err.Error())
	}

}

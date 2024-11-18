
# Project Overview

This project is a Go-based implementation of a Socket.IO client. It provides functionalities to connect to a Socket.IO server, handle events, and manage the connection lifecycle.

## Features

- Connect to a Socket.IO server using WebSocket transport.
- Emit events to the server.
- Listen for events from the server.
- Handle graceful disconnection.
- Automatic ping to keep the connection alive.

## Installation

To install the necessary dependencies, run: `go get github.com/liangsqrt/socketio-client-go`
## Usage Example

Below is a simple example of how to use the Socket.IO client in your Go project:

```go
package main

import (
	"fmt"
	"github.com/liangsqrt/socketio-client-go/socketio"
)

func main() {

	url := socketioclient.SocketIoUrl{
		Host:   "127.0.0.1",
		Port:   8080,
		Secure: false,
		Query:  map[string]string{"token": "xxxx"},
	}
	c, err := socketioclient.Dial(
		url,
		transport.GetDefaultWebsocketTransport(),
		&socketioclient.Namespace{
			Namespace: "statistic",
		},
		true,
	)
	c.On("your_event_name", func(c *socketioclient.Channel, args []interface{}) error {
		if len(args) > 0 {
			print(args)
		} else {
			print("Unexpected argument type")
		}
		return nil
	})

	if err != nil {
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
```
    
TODO:
- [x] Add automatic reconnection mechanism
- [ ] Add socket.io communication that does not rely on WebSocket, such as HTTP-based communication
- [x] Testing
- [ ] Handle the message lost when reconnecting

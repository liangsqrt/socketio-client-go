package logger

import (
	"log"
	"os"
)

func LogDebug(logLine string) {
	if os.Getenv("DEBUG") == "true" {
		log.Println(logLine)
	}
}

func LogDebugSocketIo(logLine string) {
	if os.Getenv("DEBUG_SOCKETIO") == "true" {
		log.Println(logLine)
	}
}
func LogErrorSocketIo(logLine string) {
	log.Println(logLine)
}

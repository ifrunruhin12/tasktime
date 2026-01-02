package main

import (
	"flag"

	"github.com/ifrunruhin12/tasktime/internal/client"
)

func main() {
	debugMode := flag.Bool("debug", false, "Enable debug mode logging")
	flag.Parse()

	if err := client.InitLogger(*debugMode); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer client.CloseLogger()

	client.LogInfo("Client starting", "debug_mode", *debugMode)

	c := client.New("http://localhost:8080")
	if err := c.Start(); err != nil {
		client.LogError("Client failed to start", "error", err.Error())
		panic("Client failed to start: " + err.Error())
	}
}

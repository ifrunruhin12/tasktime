package main

import (
	"flag"
	"log"

	"github.com/ifrunruhin12/tasktime/internal/client"
)

func main() {
	// Parse command line flags
	debugMode := flag.Bool("debug", false, "Enable debug mode logging")
	flag.Parse()

	// Initialize logger
	if err := client.InitLogger(*debugMode); err != nil {
		log.Printf("Failed to initialize logger: %v", err)
	}
	defer client.CloseLogger()

	// Log client startup
	client.LogInfo("Client starting", "debug_mode", *debugMode)

	c := client.New("http://localhost:8080")
	if err := c.Start(); err != nil {
		client.LogError("Client failed to start", "error", err.Error())
		log.Fatal(err)
	}
}

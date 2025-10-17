package main

import (
	"flag"
	"log"

	"github.com/ifrunruhin12/tasktime-mvp/internal/client"
)

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "TaskTime server URL")
	flag.Parse()

	c := client.New(*serverURL)
	if err := c.Start(); err != nil {
		log.Fatal(err)
	}
}
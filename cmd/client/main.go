package main

import (
	"log"

	"github.com/ifrunruhin12/tasktime/internal/client"
)

func main() {
	c := client.New("http://localhost:8080")
	if err := c.Start(); err != nil {
		log.Fatal(err)
	}
}

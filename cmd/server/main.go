package main

import (
	"flag"
	"log"

	"github.com/ifrunruhin12/tasktime/internal/server"
)

func main() {
	port := flag.String("port", "8080", "Port to run the server on")
	flag.Parse()

	srv, err := server.New()
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	if err := srv.Start(*port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

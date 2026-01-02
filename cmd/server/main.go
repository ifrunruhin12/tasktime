package main

import (
	_ "github.com/ifrunruhin12/tasktime/docs"
	"github.com/ifrunruhin12/tasktime/internal/server"
)

func main() {
	srv, err := server.New()
	if err != nil {
		panic("Failed to create server: " + err.Error())
	}

	if err := srv.Start(); err != nil {
		server.LogError("Failed to start server", "error", err.Error())
		panic("Failed to start server: " + err.Error())
	}
}

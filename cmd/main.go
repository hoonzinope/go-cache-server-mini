package main

import (
	"fmt"
	"go-cache-server-mini/internal"
)

var addr string = ":8080"

func main() {
	start()
	defer stop()
}

func start() {
	// Start the API server
	fmt.Println("Starting API server on", addr)
	cache := internal.NewCache()
	internal.StartAPIServer(addr, cache)
}

func stop() {
	// Stop the API server
	fmt.Println("Stopping API server")
}

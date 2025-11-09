package main

import (
	"fmt"
	"go-cache-server-mini/internal/api"
	"go-cache-server-mini/internal/config"
	"go-cache-server-mini/internal/core"
)

var addr string = ":8080"

func main() {
	start()
	defer stop()
}

func start() {
	// Start the API server
	fmt.Println("Starting API server on", addr)
	config, configLoadErr := config.LoadConfig()
	if configLoadErr != nil {
		fmt.Println("Error loading config:", configLoadErr)
		return
	}
	cache := core.NewCache(config.TTL.Default, config.TTL.Max)
	if config.HTTP.Enabled {
		addr = config.HTTP.Address
		api.StartAPIServer(addr, cache)
	}
}

func stop() {
	// Stop the API server
	fmt.Println("Stopping API server")
}

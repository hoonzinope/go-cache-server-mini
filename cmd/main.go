package main

import (
	"context"
	"fmt"
	"go-cache-server-mini/internal/api"
	"go-cache-server-mini/internal/config"
	"go-cache-server-mini/internal/core"
	"log"
	"sync"
)

var ctx, cancel = context.WithCancel(context.Background())

func main() {
	start()
	defer stop()
}

func start() {
	// Start the API server
	config, configLoadErr := config.LoadConfig()
	if configLoadErr != nil {
		log.Fatalf("Failed to load config: %v", configLoadErr)
	}
	cache := core.NewCache(ctx, config.TTL.Default, config.TTL.Max)
	wg := sync.WaitGroup{}
	if config.HTTP.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := config.HTTP.Address
			fmt.Println("Starting API server on", addr)
			api.StartAPIServer(ctx, addr, cache)
		}()
	}
	wg.Wait()
}

func stop() {
	// Stop the API server
	fmt.Println("Stopping API server")
	cancel()
}

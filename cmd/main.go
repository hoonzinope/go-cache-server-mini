package main

import (
	"context"
	"fmt"
	"go-cache-server-mini/internal/api"
	"go-cache-server-mini/internal/config"
	"go-cache-server-mini/internal/core"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var ctx, cancel = context.WithCancel(context.Background())
var wg = sync.WaitGroup{}

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	start()
	<-signalChan
	stop()
}

func start() {
	// Start the API server
	config, configLoadErr := config.LoadConfig("config.yml")
	if configLoadErr != nil {
		log.Fatalf("Failed to load config: %v", configLoadErr)
	}
	cache := core.NewCache(ctx, config.TTL.Default, config.TTL.Max)
	if config.HTTP.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := config.HTTP.Address
			fmt.Println("Starting API server on", addr)
			api.StartAPIServer(ctx, addr, cache)
		}()
	}
}

func stop() {
	// Stop the API server
	cancel()
	fmt.Println("Stop signal received, shutting down...")
	wg.Wait()
}

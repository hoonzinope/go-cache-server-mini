package main

import (
	"context"
	"fmt"
	"go-cache-server-mini/internal/api"
	"go-cache-server-mini/internal/config"
	"go-cache-server-mini/internal/core"
	"go-cache-server-mini/internal/distributed/adapter"
	"go-cache-server-mini/internal/distributed/router"
	"go-cache-server-mini/internal/grpc/server"
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
	defer signal.Stop(signalChan)

	errChan := make(chan error, 1)

	start(errChan)

	var shutdownErr error
	select {
	case sig := <-signalChan:
		fmt.Println("Received signal:", sig)
	case err := <-errChan:
		shutdownErr = err
		log.Printf("API server exited with error: %v\n", err)
	}

	stop()
	if shutdownErr != nil {
		os.Exit(1)
	}
}

func start(errChan chan<- error) {
	// Start the API server
	config, configLoadErr := config.LoadConfig("config.yml")
	if configLoadErr != nil {
		log.Fatalf("Failed to load config: %v", configLoadErr)
	}
	cache, createCacheErr := core.NewCache(ctx, config)
	if createCacheErr != nil {
		log.Fatalf("Failed to create cache: %v", createCacheErr)
	}

	localAdapter := adapter.NewLocalAdapter(cache)
	clusterManager := router.NewClusterManager(config.Distributed.SwarmServiceName)
	nodeRouter, err := router.NewNodeRouter(ctx, localAdapter, clusterManager, *config)
	if err != nil {
		log.Fatalf("Failed to create node router: %v", err)
	}
	cacheDistributor := router.NewDistributor(nodeRouter)
	localCacheDistributor := router.NewLocalDistributor(nodeRouter)

	// Start the API server
	if config.HTTP.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := config.HTTP.Address
			fmt.Println("Starting API server on", addr)
			var connectDistributor router.DistributorInterface
			if !config.Distributed.Enabled {
				connectDistributor = localCacheDistributor
			} else {
				connectDistributor = cacheDistributor
			}

			if err := api.StartAPIServer(ctx, addr, connectDistributor); err != nil {
				errChan <- err
			}
		}()
	}

	if config.Distributed.Enabled {
		// Start the gRPC server for inter-node communication
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := fmt.Sprintf(":%d", config.Distributed.GRPCPort)
			fmt.Println("Starting gRPC cache server on", addr)
			err := server.StartGRPCCacheServer(ctx, addr, localCacheDistributor)
			if err != nil {
				errChan <- err
			}
		}()
	}
}

func stop() {
	// Stop the API server
	cancel()
	fmt.Println("Stop signal received, shutting down...")
	wg.Wait()
}

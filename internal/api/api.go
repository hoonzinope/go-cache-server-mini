package api

import (
	"context"
	"fmt"
	"go-cache-server-mini/internal/api/handler"
	"go-cache-server-mini/internal/core"
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIServer struct {
	Addr      string
	httpSever *http.Server
	cache     core.CacheInterface
}

func StartAPIServer(ctx context.Context, addr string, cache core.CacheInterface) {
	// Implementation for starting the API server goes here
	server := APIServer{
		Addr:  addr,
		cache: cache,
	}

	httpServer := &http.Server{
		Addr:    server.Addr,
		Handler: server.router(),
	}
	server.httpSever = httpServer

	go func() {
		<-ctx.Done()
		server.Stop()
	}()

	err := httpServer.ListenAndServe()
	if err != nil {
		fmt.Printf("Failed to start API server: %v\n", err)
	}
}

func (server *APIServer) Stop() {
	// Implementation for stopping the API server goes here
	fmt.Println("Stopping API server...")
	server.httpSever.Shutdown(context.Background())
}

func (server *APIServer) router() *gin.Engine {
	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	// Ping route for health check
	server.ping(r)
	// core API routes
	// read
	server.get(r)
	server.exists(r)
	server.keys(r)
	server.ttl(r)
	// expire
	server.expire(r)
	// write
	server.set(r)
	server.del(r)
	server.flush(r)
	return r
}

func (server *APIServer) ping(r *gin.Engine) {
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
}

func (server *APIServer) set(r *gin.Engine) {
	setHandler := handler.SetHandler{
		Cache: server.cache,
	}
	r.POST("/set", setHandler.Set)
}

func (server *APIServer) get(r *gin.Engine) {
	getHandler := handler.GetHandler{
		Cache: server.cache,
	}
	r.GET("/get", getHandler.Get)
}

func (server *APIServer) del(r *gin.Engine) {
	delHandler := handler.DelHandler{
		Cache: server.cache,
	}
	r.DELETE("/del", delHandler.Del)
}

func (server *APIServer) exists(r *gin.Engine) {
	existsHandler := handler.ExistsHandler{
		Cache: server.cache,
	}
	r.GET("/exists", existsHandler.Exists)
}

func (server *APIServer) keys(r *gin.Engine) {
	keysHandler := handler.KeysHandler{
		Cache: server.cache,
	}
	r.GET("/keys", keysHandler.Keys)
}

func (server *APIServer) flush(r *gin.Engine) {
	flushHandler := handler.FlushHandler{
		Cache: server.cache,
	}
	r.POST("/flush", flushHandler.Flush)
}

func (server *APIServer) expire(r *gin.Engine) {
	expireHandler := handler.ExpireHandler{
		Cache: server.cache,
	}
	r.POST("/expire", expireHandler.Expire)
}

func (server *APIServer) ttl(r *gin.Engine) {
	ttlHandler := handler.TTLHandler{
		Cache: server.cache,
	}
	r.GET("/ttl", ttlHandler.TTL)
}

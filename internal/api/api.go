package api

import (
	"go-cache-server-mini/internal/api/handler"
	"go-cache-server-mini/internal/core"
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIServer struct {
	Addr  string
	cache *core.Cache
}

func StartAPIServer(addr string, cache *core.Cache) {
	// Implementation for starting the API server goes here
	server := APIServer{
		Addr:  addr,
		cache: cache,
	}
	http.ListenAndServe(server.Addr, server.router())
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

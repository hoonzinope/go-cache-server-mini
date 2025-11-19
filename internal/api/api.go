package api

import (
	"context"
	"errors"
	"fmt"
	"go-cache-server-mini/internal/api/handler"
	"go-cache-server-mini/internal/distributed/router"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type APIServer struct {
	Addr        string
	httpSever   *http.Server
	Distributor router.DistributorInterface
}

func StartAPIServer(ctx context.Context, addr string, distributor router.DistributorInterface) error {
	// Implementation for starting the API server goes here
	server := APIServer{
		Addr:        addr,
		Distributor: distributor,
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

	fmt.Printf("Starting API server at %s\n", server.Addr)
	err := httpServer.ListenAndServe()
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("failed to start API server: %w", err)
	}
	return nil
}

func (server *APIServer) Stop() {
	// Implementation for stopping the API server goes here
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.httpSever.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("API server shutdown error: %v\n", err)
	} else {
		fmt.Println("API server stopped gracefully.")
	}
}

func (server *APIServer) router() *gin.Engine {
	r := gin.New()
	// Middleware
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
	server.persist(r)
	server.incr(r)
	server.decr(r)
	// extra
	server.setNX(r)
	server.getSet(r)
	server.mGet(r)
	server.mSet(r)
	return r
}

func (server *APIServer) ping(r *gin.Engine) {
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
}

func (server *APIServer) set(r *gin.Engine) {
	setHandler := handler.SetHandler{
		Cache: server.Distributor,
	}
	r.POST("/set", setHandler.Set)
}

func (server *APIServer) get(r *gin.Engine) {
	getHandler := handler.GetHandler{
		Cache: server.Distributor,
	}
	r.GET("/get", getHandler.Get)
}

func (server *APIServer) del(r *gin.Engine) {
	delHandler := handler.DelHandler{
		Cache: server.Distributor,
	}
	r.DELETE("/del", delHandler.Del)
}

func (server *APIServer) exists(r *gin.Engine) {
	existsHandler := handler.ExistsHandler{
		Cache: server.Distributor,
	}
	r.GET("/exists", existsHandler.Exists)
}

func (server *APIServer) keys(r *gin.Engine) {
	keysHandler := handler.KeysHandler{
		Cache: server.Distributor,
	}
	r.GET("/keys", keysHandler.Keys)
}

func (server *APIServer) flush(r *gin.Engine) {
	flushHandler := handler.FlushHandler{
		Cache: server.Distributor,
	}
	r.POST("/flush", flushHandler.Flush)
}

func (server *APIServer) expire(r *gin.Engine) {
	expireHandler := handler.ExpireHandler{
		Cache: server.Distributor,
	}
	r.POST("/expire", expireHandler.Expire)
}

func (server *APIServer) ttl(r *gin.Engine) {
	ttlHandler := handler.TTLHandler{
		Cache: server.Distributor,
	}
	r.GET("/ttl", ttlHandler.TTL)
}

func (server *APIServer) persist(r *gin.Engine) {
	persistHandler := handler.PersistHandler{
		Cache: server.Distributor,
	}
	r.POST("/persist", persistHandler.Persist)
}

func (server *APIServer) incr(r *gin.Engine) {
	incrHandler := handler.IncrHandler{
		Cache: server.Distributor,
	}
	r.POST("/incr", incrHandler.Incr)
}

func (server *APIServer) decr(r *gin.Engine) {
	decrHandler := handler.DecrHandler{
		Cache: server.Distributor,
	}
	r.POST("/decr", decrHandler.Decr)
}

func (server *APIServer) setNX(r *gin.Engine) {
	setNXHandler := handler.SetNXHandler{
		Cache: server.Distributor,
	}
	r.POST("/setnx", setNXHandler.SetNX)
}

func (server *APIServer) getSet(r *gin.Engine) {
	getSetHandler := handler.GetSetHandler{
		Cache: server.Distributor,
	}
	r.POST("/getset", getSetHandler.GetSet)
}

func (server *APIServer) mGet(r *gin.Engine) {
	mGetHandler := handler.MGetHandler{
		Cache: server.Distributor,
	}
	r.POST("/mget", mGetHandler.MGet)
}

func (server *APIServer) mSet(r *gin.Engine) {
	mSetHandler := handler.MSetHandler{
		Cache: server.Distributor,
	}
	r.POST("/mset", mSetHandler.MSet)
}

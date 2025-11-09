package internal

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIServer struct {
	Addr  string
	cache *Cache
}

func StartAPIServer(addr string, cache *Cache) {
	// Implementation for starting the API server goes here
	server := APIServer{
		Addr:  addr,
		cache: cache,
	}
	http.ListenAndServe(server.Addr, server.router())
}

func (server *APIServer) router() *gin.Engine {
	r := gin.Default()
	r.POST("/set", server.set)
	r.GET("/get", server.get)
	return r
}

type SetRequest struct {
	Key   string          `json:"key" binding:"required"`
	Value json.RawMessage `json:"value" binding:"required"`
}

func (server *APIServer) set(c *gin.Context) {
	var req SetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println("Error binding JSON:", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBadRequest.Error()})
		return
	}
	setErr := server.cache.Set(req.Key, req.Value)
	if setErr != nil {
		fmt.Println("Error setting cache:", setErr.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": ErrServer.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

type GetRequest struct {
	Key string `form:"key" binding:"required"`
}

type GetResponse struct {
	Value json.RawMessage `json:"value"`
}

func (server *APIServer) get(c *gin.Context) {
	var req GetRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		fmt.Println("Error binding query:", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBadRequest.Error()})
		return
	}
	value, ok := server.cache.Get(req.Key)
	if !ok {
		fmt.Println("Error getting cache:", ErrNotFound.Error())
		c.JSON(http.StatusNotFound, gin.H{"error": ErrNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, GetResponse{Value: value})
}

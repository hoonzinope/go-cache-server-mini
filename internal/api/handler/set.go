package handler

import (
	"fmt"
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/core"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type SetHandler struct {
	Cache core.CacheInterface
}

func (h *SetHandler) Set(c *gin.Context) {
	var req dto.SetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println("Error binding JSON:", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	if req.TTL < 0 {
		req.TTL = 0
	}
	ttl := time.Duration(req.TTL) * time.Second
	setErr := h.Cache.Set(req.Key, req.Value, ttl)
	if setErr != nil {
		fmt.Println("Error setting cache:", setErr.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": internal.ErrServer.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

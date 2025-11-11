package handler

import (
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/core"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type MSetHandler struct {
	Cache core.CacheInterface
}

func (h *MSetHandler) MSet(c *gin.Context) {
	var req dto.MSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	ttl := time.Duration(req.TTL) * time.Second
	kv := make(map[string][]byte)
	for key, value := range req.KV {
		kv[key] = value
	}
	setErr := h.Cache.MSet(kv, ttl)
	if setErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": internal.ErrServer.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

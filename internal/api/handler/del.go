package handler

import (
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/core"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DelHandler struct {
	Cache core.CacheInterface
}

func (h *DelHandler) Del(c *gin.Context) {
	var req dto.DelRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}

	// Check if the key exists in the cache
	if !h.Cache.Exists(req.Key) {
		c.JSON(http.StatusNotFound, gin.H{"error": internal.ErrNotFound.Error()})
		return
	}

	delErr := h.Cache.Del(req.Key)
	if delErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": internal.ErrServer.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

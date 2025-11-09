package handler

import (
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/core"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TTLHandler struct {
	Cache core.CacheInterface
}

func (h *TTLHandler) TTL(c *gin.Context) {
	var req dto.TTLRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}

	ttl, exists := h.Cache.TTL(req.Key)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": internal.ErrNotFound.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ttl": int64(ttl.Seconds())})
}

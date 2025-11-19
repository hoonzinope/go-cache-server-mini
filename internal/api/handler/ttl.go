package handler

import (
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/distributed/router"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TTLHandler struct {
	Cache router.DistributorInterface
}

func (h *TTLHandler) TTL(c *gin.Context) {
	var req dto.KeyRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}

	ttl, exists, err := h.Cache.TTL(req.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": internal.ErrNotFound.Error()})
		return
	}
	if ttl < 0 {
		c.JSON(http.StatusOK, gin.H{"ttl": -1})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ttl": int64(ttl.Seconds())})
}

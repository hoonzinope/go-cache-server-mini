package handler

import (
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/distributed/router"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type SetNXHandler struct {
	Cache router.DistributorInterface
}

func (h *SetNXHandler) SetNX(c *gin.Context) {
	var req dto.SetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	ttl := time.Duration(req.TTL) * time.Second
	success, setErr := h.Cache.SetNX(req.Key, req.Value, ttl)
	if setErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": internal.ErrServer.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": success})
}

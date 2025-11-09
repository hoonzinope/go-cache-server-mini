package handler

import (
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/core"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ExistsHandler struct {
	Cache core.CacheInterface
}

func (h *ExistsHandler) Exists(c *gin.Context) {
	var req dto.ExistsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	exists := h.Cache.Exists(req.Key)
	c.JSON(http.StatusOK, gin.H{"exists": exists})
}

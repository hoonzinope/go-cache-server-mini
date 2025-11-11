package handler

import (
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/core"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GetSetHandler struct {
	Cache core.CacheInterface
}

func (h *GetSetHandler) GetSet(c *gin.Context) {
	var req dto.GetSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	oldValue, getSetErr := h.Cache.GetSet(req.Key, req.Value)
	if getSetErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": internal.ErrServer.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.ValueResponse{Value: oldValue})
}

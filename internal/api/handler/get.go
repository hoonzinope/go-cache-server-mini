package handler

import (
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/core"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GetHandler struct {
	Cache core.CacheInterface
}

func (h *GetHandler) Get(c *gin.Context) {
	var req dto.GetRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Printf("Error binding query: %v", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	value, ok := h.Cache.Get(req.Key)
	if !ok {
		log.Printf("Error getting cache : %v for key: %s", internal.ErrNotFound.Error(), req.Key)
		c.JSON(http.StatusNotFound, gin.H{"error": internal.ErrNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.GetResponse{Value: value})
}

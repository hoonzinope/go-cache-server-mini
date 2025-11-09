package handler

import (
	"fmt"
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/core"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GetHandler struct {
	Cache core.CacheInterface
}

func (h *GetHandler) Get(c *gin.Context) {
	var req dto.GetRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		fmt.Println("Error binding query:", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	value, ok := h.Cache.Get(req.Key)
	if !ok {
		fmt.Println("Error getting cache:", internal.ErrNotFound.Error())
		c.JSON(http.StatusNotFound, gin.H{"error": internal.ErrNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.GetResponse{Value: value})
}

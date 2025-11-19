package handler

import (
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/distributed/router"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GetHandler struct {
	Cache router.DistributorInterface
}

func (h *GetHandler) Get(c *gin.Context) {
	var req dto.KeyRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Printf("Error binding query: %v", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	value, ok, err := h.Cache.Get(req.Key)
	if err != nil {
		log.Printf("Error getting cache : %v for key: %s", err.Error(), req.Key)
		c.JSON(http.StatusInternalServerError, gin.H{"error": internal.ErrServer.Error()})
		return
	}
	if !ok {
		log.Printf("Error getting cache : %v for key: %s", internal.ErrNotFound.Error(), req.Key)
		c.JSON(http.StatusNotFound, gin.H{"error": internal.ErrNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, dto.ValueResponse{Value: value})
}

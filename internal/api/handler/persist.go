package handler

import (
	"errors"
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/distributed/router"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PersistHandler struct {
	Cache router.DistributorInterface
}

func (h *PersistHandler) Persist(c *gin.Context) {
	var req dto.KeyRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Printf("Error binding query: %v", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	if err := h.Cache.Persist(req.Key); err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": internal.ErrNotFound.Error()})
			return
		}
		log.Printf("Error persisting cache: %v for key: %s", err.Error(), req.Key)
		c.JSON(http.StatusInternalServerError, gin.H{"error": internal.ErrServer.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

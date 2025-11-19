package handler

import (
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/distributed/router"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DelHandler struct {
	Cache router.DistributorInterface
}

func (h *DelHandler) Del(c *gin.Context) {
	var req dto.KeyRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	delErr := h.Cache.Del(req.Key)
	if delErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": internal.ErrServer.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

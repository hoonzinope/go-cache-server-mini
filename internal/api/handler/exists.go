package handler

import (
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/distributed/router"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ExistsHandler struct {
	Cache router.DistributorInterface
}

func (h *ExistsHandler) Exists(c *gin.Context) {
	var req dto.KeyRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	exists, err := h.Cache.Exists(req.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"exists": exists})
}

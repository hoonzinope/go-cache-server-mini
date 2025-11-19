package handler

import (
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/distributed/router"
	"net/http"

	"github.com/gin-gonic/gin"
)

type FlushHandler struct {
	Cache router.DistributorInterface
}

func (h *FlushHandler) Flush(c *gin.Context) {
	flushErr := h.Cache.Flush()
	if flushErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": internal.ErrServer.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

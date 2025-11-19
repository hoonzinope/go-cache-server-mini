package handler

import (
	"go-cache-server-mini/internal/distributed/router"
	"net/http"

	"github.com/gin-gonic/gin"
)

type KeysHandler struct {
	Cache router.DistributorInterface
}

func (h *KeysHandler) Keys(c *gin.Context) {
	keys, err := h.Cache.Keys()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

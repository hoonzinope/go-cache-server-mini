package handler

import (
	"go-cache-server-mini/internal/core"
	"net/http"

	"github.com/gin-gonic/gin"
)

type KeysHandler struct {
	Cache core.CacheInterface
}

func (h *KeysHandler) Keys(c *gin.Context) {
	keys := h.Cache.Keys()
	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

package handler

import (
	"errors"
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/core"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ExpireHandler struct {
	Cache core.CacheInterface
}

func (h *ExpireHandler) Expire(c *gin.Context) {
	var expireReq dto.ExpireRequest
	if err := c.ShouldBindJSON(&expireReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	expireErr := h.Cache.Expire(expireReq.Key, time.Duration(expireReq.TTL)*time.Second)
	if expireErr != nil {
		if errors.Is(expireErr, internal.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": internal.ErrNotFound.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": internal.ErrServer.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

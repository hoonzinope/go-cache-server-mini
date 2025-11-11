package handler

import (
	"encoding/json"
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/core"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MGetHandler struct {
	Cache core.CacheInterface
}

func (h *MGetHandler) MGet(c *gin.Context) {
	var req dto.MGetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	kv := h.Cache.MGet(req.Keys)
	var res dto.MGetResponse = dto.MGetResponse{KV: make(map[string]json.RawMessage, len(kv))}
	for key, value := range kv {
		res.KV[key] = json.RawMessage(value)
	}
	c.JSON(http.StatusOK, res)
}

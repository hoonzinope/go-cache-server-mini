package handler

import (
	"encoding/json"
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/distributed/router"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MGetHandler struct {
	Cache router.DistributorInterface
}

func (h *MGetHandler) MGet(c *gin.Context) {
	var req dto.MGetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	kv, err := h.Cache.MGet(req.Keys)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var res dto.MGetResponse = dto.MGetResponse{KV: make(map[string]json.RawMessage, len(kv))}
	for key, value := range kv {
		res.KV[key] = json.RawMessage(value)
	}
	c.JSON(http.StatusOK, res)
}

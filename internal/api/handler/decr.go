package handler

import (
	"encoding/json"
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/api/dto"
	"go-cache-server-mini/internal/distributed/router"
	"go-cache-server-mini/internal/util"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DecrHandler struct {
	Cache router.DistributorInterface
}

func (h *DecrHandler) Decr(c *gin.Context) {
	var req dto.KeyRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Printf("Error binding query: %v", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrBadRequest.Error()})
		return
	}
	newValue, decrErr := h.Cache.Decr(req.Key)
	if decrErr != nil {
		if decrErr == internal.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": internal.ErrNotFound.Error()})
			return
		}
		log.Printf("Error decrementing cache: %v for key: %s", decrErr.Error(), req.Key)
		c.JSON(http.StatusInternalServerError, gin.H{"error": internal.ErrServer.Error()})
		return
	}
	var res dto.ValueResponse
	res.Value = json.RawMessage(util.Int64ToBytes(newValue))
	c.JSON(http.StatusOK, res)
}

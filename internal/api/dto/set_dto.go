package dto

import (
	"encoding/json"
)

type SetRequest struct {
	Key   string          `json:"key" binding:"required"`
	Value json.RawMessage `json:"value" binding:"required"`
	TTL   int64           `json:"ttl" binding:"omitempty"`
}

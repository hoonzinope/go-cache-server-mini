package dto

import "encoding/json"

type KeyRequest struct {
	Key string `form:"key" binding:"required"`
}

type ExpireRequest struct {
	Key string `json:"key" binding:"required"`
	TTL int64  `json:"ttl" binding:"required"`
}

type ValueResponse struct {
	Value json.RawMessage `json:"value"`
}

type SetRequest struct {
	Key   string          `json:"key" binding:"required"`
	Value json.RawMessage `json:"value" binding:"required"`
	TTL   int64           `json:"ttl" binding:"omitempty"`
}

type GetSetRequest struct {
	Key   string          `json:"key" binding:"required"`
	Value json.RawMessage `json:"value" binding:"required"`
}

type MGetRequest struct {
	Keys []string `json:"keys" binding:"required"`
}

type MGetResponse struct {
	KV map[string]json.RawMessage `json:"kv"`
}

type MSetRequest struct {
	KV  map[string]json.RawMessage `json:"kv" binding:"required"`
	TTL int64                      `json:"ttl" binding:"omitempty"`
}

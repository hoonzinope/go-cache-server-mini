package dto

import "encoding/json"

type GetRequest struct {
	Key string `form:"key" binding:"required"`
}

type GetResponse struct {
	Value json.RawMessage `json:"value"`
}

package dto

type ExpireRequest struct {
	Key string `json:"key" binding:"required"`
	TTL int64  `json:"ttl" binding:"required"`
}

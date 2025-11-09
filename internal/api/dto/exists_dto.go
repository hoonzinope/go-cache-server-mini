package dto

type ExistsRequest struct {
	Key string `form:"key" binding:"required"`
}

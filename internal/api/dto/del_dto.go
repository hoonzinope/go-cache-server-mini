package dto

type DelRequest struct {
	Key string `form:"key" binding:"required"`
}

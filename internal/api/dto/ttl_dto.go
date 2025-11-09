package dto

type TTLRequest struct {
	Key string `form:"key" binding:"required"`
}

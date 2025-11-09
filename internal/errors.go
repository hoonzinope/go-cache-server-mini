package internal

import "errors"

var (
	ErrBadRequest = errors.New("bad request")
	ErrNotFound   = errors.New("key not found in cache")
	ErrServer     = errors.New("internal server error")
)

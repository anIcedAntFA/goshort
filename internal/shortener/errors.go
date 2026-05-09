package shortener

import "errors"

// Sentinel errors returned by the shortener service.
var (
	ErrNotFound       = errors.New("not found")
	ErrExpired        = errors.New("url expired")
	ErrAliasTaken     = errors.New("alias already taken")
	ErrReservedPath   = errors.New("alias is a reserved path")
	ErrInvalidURL     = errors.New("invalid url")
	ErrInvalidAlias   = errors.New("invalid alias")
	ErrInvalidExpires = errors.New("invalid expires_in duration")
	ErrBatchTooLarge  = errors.New("batch exceeds maximum of 50 items")
	ErrBatchEmpty     = errors.New("batch must contain at least one item")
)

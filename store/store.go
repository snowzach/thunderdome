package store

import (
	"errors"
)

// ErrNotFound is a standard no found error
var (
	ErrNotFound          = errors.New("Not Found")
	ErrInsufficientFunds = errors.New("Insufficient Funds")
	ErrRequestExpired    = errors.New("Request Expired")
)

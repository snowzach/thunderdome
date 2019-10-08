package store

import (
	"errors"
)

// ErrNotFound is a standard no found error
var (
	ErrNotFound          = errors.New("not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrRequestExpired    = errors.New("request expired")
)

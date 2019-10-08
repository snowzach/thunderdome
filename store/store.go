package store

import (
	"errors"
)

// ErrNotFound is a standard no found error
var (
	ErrNotFound = errors.New("not found")
)

package store

import (
	"errors"
	"time"
)

// ErrNotFound is a standard no found error
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exits")
)

type DistCache interface {
	Set(bucket string, key string, value interface{}, expires time.Duration) error
	Exists(bucket string, key string) (bool, error)
	GetScan(bucket string, key string, dest interface{}) error
	GetBytes(bucket string, key string) ([]byte, error)
	Del(bucket string, key string) error
	Clear(bucket string) error
}

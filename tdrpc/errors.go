package tdrpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrSigVerficationFailed   = status.Errorf(codes.Unauthenticated, "signature verification failed")
	ErrInvalidSig             = status.Errorf(codes.Unauthenticated, "signature invalid")
	ErrInvalidPubKey          = status.Errorf(codes.Unauthenticated, "invalid public key string")
	ErrInvalidTimestamp       = status.Errorf(codes.Unauthenticated, "invalid timestamp")
	ErrInvalidTimestampOffset = status.Errorf(codes.Unauthenticated, "invalid timestamp offset")
	ErrInvalidLogin           = status.Errorf(codes.Unauthenticated, "invalid login")
	ErrAccountLocked          = status.Errorf(codes.PermissionDenied, "account is locked")
	ErrServiceUnavailable     = status.Errorf(codes.Unavailable, "service unavailable")
	ErrRequestExpired         = status.Errorf(codes.InvalidArgument, "request is expired")
	ErrRequestAlreadyPaid     = status.Errorf(codes.InvalidArgument, "request already paid")
	ErrInsufficientFunds      = status.Errorf(codes.InvalidArgument, "insufficient funds")
	ErrNoRouteFound           = status.Errorf(codes.Internal, "network route not found")
	ErrNotFound               = status.Errorf(codes.NotFound, "not found")
)
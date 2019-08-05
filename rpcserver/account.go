package rpcserver

import (
	"context"

	emptypb "github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// GetAccount will return the account
func (s *RPCServer) GetAccount(ctx context.Context, _ *emptypb.Empty) (*tdrpc.Account, error) {

	// The authentication function will upsert the account and include it in the request context
	account := getAccount(ctx)

	// This is never really possible, but just for sanities sake
	if account == nil {
		return nil, status.Errorf(codes.Internal, "Missing Account")
	}

	return account, nil

}

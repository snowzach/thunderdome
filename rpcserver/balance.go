package rpcserver

import (
	"context"

	emptypb "github.com/golang/protobuf/ptypes/empty"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// Balance will provide the users balance
func (s *RPCServer) Balance(ctx context.Context, _ *emptypb.Empty) (*tdrpc.BalanceResponse, error) {

	// Get the authenticated user from the context
	user := getUser(ctx)
	if user == nil {
		return nil, grpc.Errorf(codes.Internal, "Did not fetch user from context")
	}

	// Return their balance
	return &tdrpc.BalanceResponse{
		Balance: user.Balance,
	}, nil

}

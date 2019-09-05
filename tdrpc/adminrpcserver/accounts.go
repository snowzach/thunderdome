package adminrpcserver

import (
	"context"

	"git.coinninja.net/backend/thunderdome/tdrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *adminRPCServer) Accounts(ctx context.Context, request *tdrpc.AdminAccountsRequest) (*tdrpc.AdminAccountsResponse, error) {

	if request.Offset == 0 {
		request.Offset = -1
	}

	if request.Filter == nil {
		request.Filter = make(map[string]string)
	}

	accounts, err := s.store.GetAccounts(ctx, request.Filter, int(request.Offset), int(request.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error on GetAccounts: %v", err)
	}

	return &tdrpc.AdminAccountsResponse{
		Accounts: accounts,
	}, nil

}

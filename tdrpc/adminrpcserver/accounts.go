package adminrpcserver

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func (s *adminRPCServer) ListAccounts(ctx context.Context, request *tdrpc.AdminAccountsRequest) (*tdrpc.AdminAccountsResponse, error) {

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

func (s *adminRPCServer) GetAccount(ctx context.Context, request *tdrpc.AdminGetAccountRequest) (*tdrpc.Account, error) {

	request.Id = strings.Replace(request.Id, "-", ":", -1)

	var account *tdrpc.Account
	var err error

	if request.Id != "" {
		account, err = s.store.GetAccountByID(ctx, request.Id)
	} else if request.Address != "" {
		account, err = s.store.GetAccountByAddress(ctx, request.Address)
	} else {
		return nil, status.Errorf(codes.InvalidArgument, "You must specify id or address: %v", err)
	}

	if err == store.ErrNotFound {
		return nil, status.Errorf(codes.NotFound, "account not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not fetch account: %v", err)
	}

	return account, nil

}

func (s *adminRPCServer) UpdateAccount(ctx context.Context, request *tdrpc.AdminUpdateAccountRequest) (*tdrpc.Account, error) {

	request.Id = strings.Replace(request.Id, "-", ":", -1)

	if request.Id == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid id")
	}

	account, err := s.store.GetAccountByID(ctx, request.Id)
	if err == store.ErrNotFound {
		return nil, status.Errorf(codes.NotFound, "account not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not fetch account: %v", err)
	}

	// The only thing we will allow updating is the locked value
	account.Locked = request.Locked

	account, err = s.store.SaveAccount(ctx, account)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not update account: %v", err)
	}

	return account, nil

}

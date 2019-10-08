package tdrpcserver

import (
	"context"

	emptypb "github.com/golang/protobuf/ptypes/empty"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// GetAccount will return the account
func (s *tdRPCServer) GetAccount(ctx context.Context, _ *emptypb.Empty) (*tdrpc.Account, error) {

	// The authentication function will upsert the account and include it in the request context
	account := getAccount(ctx)
	if account == nil {
		return nil, tdrpc.ErrNotFound
	}

	// If the account is locked, don't reveal the address
	if account.Locked {
		account.Address = ""
	}

	return account, nil

}

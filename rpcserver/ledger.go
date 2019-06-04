package rpcserver

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// Ledger will return the ledger for a user
func (s *RPCServer) Ledger(ctx context.Context, request *tdrpc.LedgerRequest) (*tdrpc.LedgerResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, status.Errorf(codes.Internal, "Missing Account")
	}

	lrs, err := s.store.GetLedger(ctx, account.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error on GetLedger: %v", err)
	}

	return &tdrpc.LedgerResponse{
		Ledger: lrs,
	}, nil

}

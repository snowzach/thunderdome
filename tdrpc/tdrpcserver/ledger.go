package tdrpcserver

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// Ledger will return the ledger for a user
func (s *tdRPCServer) Ledger(ctx context.Context, request *tdrpc.LedgerRequest) (*tdrpc.LedgerResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, status.Errorf(codes.Internal, "Missing Account")
	}

	// Ensure after has a value
	var after time.Time
	if request.After != nil {
		after = *request.After
	}

	if request.Filter == nil {
		request.Filter = make(map[string]string)
	}

	// Always force this users account id
	request.Filter["account_id"] = account.Id

	// If not specified, don't show hidden entries
	if _, ok := request.Filter["hidden"]; !ok {
		request.Filter["hidden"] = "false"
	}

	lrs, err := s.store.GetLedger(ctx, request.Filter, after, int(request.Offset), int(request.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error on GetLedger: %v", err)
	}

	if lrs == nil {
		lrs = []*tdrpc.LedgerRecord{}

	}

	return &tdrpc.LedgerResponse{
		Ledger: lrs,
	}, nil

}

package adminrpcserver

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// Ledger will return the ledger for a user
func (s *adminRPCServer) Ledger(ctx context.Context, request *tdrpc.LedgerRequest) (*tdrpc.LedgerResponse, error) {

	// Ensure after has a value
	var after time.Time
	if request.After != nil {
		after = *request.After
	}

	if request.Filter == nil {
		request.Filter = make(map[string]string)
	}

	// If not specified, don't show hidden entries
	if _, ok := request.Filter["hidden"]; !ok {
		request.Filter["hidden"] = "false"
	}

	lrs, err := s.store.GetLedger(ctx, request.Filter, after, int(request.Offset), int(request.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error on GetLedger: %v", err)
	}

	return &tdrpc.LedgerResponse{
		Ledger: lrs,
	}, nil

}

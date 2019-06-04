package rpcserver

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// Create creates a payment request for the current user
func (s *RPCServer) Create(ctx context.Context, request *tdrpc.CreateRequest) (*tdrpc.CreateResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, status.Errorf(codes.Internal, "Missing Account")
	}

	if request.Value < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid Value")
	}

	if request.Expires != 0 && (request.Expires < 300 || request.Expires > 604800) {
		return nil, status.Errorf(codes.InvalidArgument, "Expires cannot be less than 300 or greater than 604800")
	}

	// Create the invoice
	invoice, err := s.lclient.AddInvoice(ctx, &lnrpc.Invoice{
		Memo:   request.Memo,
		Value:  request.Value,
		Expiry: request.Expires,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not AddInvoice: %v", err)
	}

	// Put it in the ledger
	expiresAt := time.Now().UTC().Add(86400 * time.Second)
	err = s.store.ProcessLedgerRecord(ctx, &tdrpc.LedgerRecord{
		Id:        hex.EncodeToString(invoice.RHash),
		AccountId: account.Id,
		ExpiresAt: &expiresAt,
		Status:    tdrpc.PENDING,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.IN,
		Value:     request.Value,
		AddIndex:  invoice.AddIndex,
		Memo:      request.Memo,
		Request:   invoice.PaymentRequest,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not UpsertLedgerRecord: %v", err)
	}

	// Return the payment request
	return &tdrpc.CreateResponse{
		Request: invoice.PaymentRequest,
	}, nil

}

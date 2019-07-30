package rpcserver

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
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
	if request.Expires == 0 {
		request.Expires = config.GetInt64("tdome.default_request_expires")
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

	// Get the expires time
	expiresAt := time.Now().UTC().Add(time.Duration(request.Expires) * time.Second)

	// Put it in the ledger
	err = s.store.ProcessLedgerRecord(ctx, &tdrpc.LedgerRecord{
		Id:        hex.EncodeToString(invoice.RHash),
		AccountId: account.Id,
		ExpiresAt: &expiresAt,
		Status:    tdrpc.PENDING,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.IN,
		Value:     request.Value,
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

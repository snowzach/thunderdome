package tdrpcserver

import (
	"context"
	"encoding/hex"
	"time"

	emptypb "github.com/golang/protobuf/ptypes/empty"
	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// Create creates a payment request for the current user
func (s *tdRPCServer) Create(ctx context.Context, request *tdrpc.CreateRequest) (*tdrpc.CreateResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, status.Errorf(codes.Internal, "Missing Account")
	}

	if request.Value < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid Value")
	}

	if request.Expires != 0 && (request.Expires < 300 || request.Expires > 7776000) {
		return nil, status.Errorf(codes.InvalidArgument, "Expires cannot be less than 300 or greater than 7776000")
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

// CreateGenerated makes a payment request with no value. If one exists already, it will be returned.
func (s *tdRPCServer) CreateGenerated(ctx context.Context, _ *emptypb.Empty) (*tdrpc.CreateResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, status.Errorf(codes.Internal, "Missing Account")
	}

	// See if we already have an existing invoice
	lr, err := s.store.GetActiveGeneratedLightningLedgerRequest(ctx, account.Id)
	if err == nil {
		// Found one, return it
		return &tdrpc.CreateResponse{
			Request: lr.Request,
		}, nil
	} else if err != nil && err != store.ErrNotFound {
		// Some other error
		return nil, status.Errorf(codes.Internal, "Could not get record: %v", err)
	}

	expirationSeconds := config.GetInt64("tdome.create_generic_expiration")
	expiresAt := time.Now().UTC().Add(time.Duration(expirationSeconds) * time.Second)
	memo := "Generated request"

	// Create the invoice
	invoice, err := s.lclient.AddInvoice(ctx, &lnrpc.Invoice{
		Memo:   memo,
		Value:  0,
		Expiry: expirationSeconds,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not AddInvoice: %v", err)
	}

	// Put it in the ledger
	err = s.store.ProcessLedgerRecord(ctx, &tdrpc.LedgerRecord{
		Id:        hex.EncodeToString(invoice.RHash),
		AccountId: account.Id,
		ExpiresAt: &expiresAt,
		Status:    tdrpc.PENDING,
		Generated: true,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.IN,
		Value:     0,
		AddIndex:  invoice.AddIndex,
		Memo:      memo,
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

package rpcserver

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// DecodePayReq passes through decoding the pay request
func (s *RPCServer) Decode(ctx context.Context, request *tdrpc.DecodeRequest) (*tdrpc.DecodeResponse, error) {

	// Decode and return the PayRequest
	pr, err := s.lclient.DecodePayReq(ctx, &lnrpc.PayReqString{PayReq: request.Request})
	if err != nil {
		return nil, err
	}

	// Convert it to our type
	return &tdrpc.DecodeResponse{
		Destination:     pr.Destination,
		PaymentHash:     pr.PaymentHash,
		NumSatoshis:     pr.NumSatoshis,
		Timestamp:       pr.Timestamp,
		Expiry:          pr.Expiry,
		Description:     pr.Description,
		DescriptionHash: pr.DescriptionHash,
		FallbackAddr:    pr.FallbackAddr,
		CltvExpiry:      pr.CltvExpiry,
		RouteHints:      []*tdrpc.RouteHint{}, // TODO: Decode route hints
	}, err

}

// Create creates a payment request for the current user
func (s *RPCServer) Create(ctx context.Context, request *tdrpc.CreateRequest) (*tdrpc.CreateResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, grpc.Errorf(codes.Internal, "Missing Account")
	}

	// Create the invoice
	invoice, err := s.lclient.AddInvoice(ctx, &lnrpc.Invoice{
		Memo:   request.Memo,
		Value:  request.Value,
		Expiry: 86400,
	})
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Could not AddInvoice: %v", err)
	}

	// Put it in the ledger
	expiresAt := time.Now().UTC().Add(86400 * time.Second)
	err = s.rpcStore.UpsertLedgerRecord(ctx, &tdrpc.LedgerRecord{
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
		return nil, grpc.Errorf(codes.Internal, "Could not UpsertLedgerRecord: %v", err)
	}

	// Return the payment request
	return &tdrpc.CreateResponse{
		Request: invoice.PaymentRequest,
	}, nil

}

// Pay will pay a payment request
func (s *RPCServer) Pay(ctx context.Context, request *tdrpc.PayRequest) (*tdrpc.PayResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, grpc.Errorf(codes.Internal, "Missing Account")
	}

	// Decode the Request
	pr, err := s.lclient.DecodePayReq(ctx, &lnrpc.PayReqString{PayReq: request.Request})
	if err != nil {
		return nil, err
	}

	// Check for expiration
	expiresAt := time.Unix(pr.Expiry, 0)
	if time.Now().Sub(expiresAt) <= 0 {
		return nil, grpc.Errorf(codes.InvalidArgument, "Request is expired")
	}

	// If no value specified, pay the amount in the PayReq
	if request.Value == 0 {
		request.Value = pr.NumSatoshis
	}

	// Build the ledger record
	lr := &tdrpc.LedgerRecord{
		Id:        pr.PaymentHash,
		AccountId: account.Id,
		ExpiresAt: &expiresAt,
		Status:    tdrpc.PENDING,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.OUT,
		Value:     request.Value,
		Memo:      pr.Description,
		Request:   request.Request,
	}

	// Save the initial state
	err = s.rpcStore.UpsertLedgerRecord(ctx, lr)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Could not UpsertLedgerRecord: %v", err)
	}

	// Send the payment
	response, err := s.lclient.SendPaymentSync(ctx, &lnrpc.SendRequest{
		Amt:            request.Value,
		PaymentRequest: request.Request,
	})
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Could not SendPaymentSync: %v", err)
	} else if response.PaymentError != "" {
		lr.Status = tdrpc.FAILED
	} else {
		lr.Status = tdrpc.COMPLETED
	}

	// Update the status and the balance
	err = s.rpcStore.UpsertLedgerRecord(ctx, lr)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Could not UpsertLedgerRecord: %v", err)
	}

	return &tdrpc.PayResponse{
		Error: response.GetPaymentError(),
	}, nil
}

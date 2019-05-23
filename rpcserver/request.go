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
	err = s.rpcStore.ProcessLedgerRecord(ctx, &tdrpc.LedgerRecord{
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
	expiresAt := time.Unix(pr.Timestamp+pr.Expiry, 0)
	if time.Now().UTC().After(expiresAt) {
		return nil, grpc.Errorf(codes.InvalidArgument, "Request is expired")
	}

	// Check for zero amount
	if pr.NumSatoshis == 0 && request.Value == 0 {
		return nil, grpc.Errorf(codes.InvalidArgument, "Amount must be specified when paying a zero amount invoice")
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

	// Save the initial state - will do some sanity checking as well
	err = s.rpcStore.ProcessLedgerRecord(ctx, lr)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", err)
	}

	// If this is a payment to someone else using this service, we transfer the balance internally
	if pr.Destination == s.myPubKey {

		// This is an internal payment, process the record
		lr, err = s.rpcStore.ProcessInternal(ctx, pr.PaymentHash)
		if err != nil {
			return nil, grpc.Errorf(codes.Internal, "%v", err)
		}

		return &tdrpc.PayResponse{
			Result: lr,
		}, nil

	}

	// Send the payment
	response, err := s.lclient.SendPaymentSync(ctx, &lnrpc.SendRequest{
		Amt:            request.Value,
		PaymentRequest: request.Request,
	})
	if err != nil || (response != nil && response.PaymentError != "") {
		lr.Status = tdrpc.FAILED
		if response.PaymentError != "" {
			lr.Error = response.PaymentError
		} else {
			lr.Error = err.Error()
		}
	} else {
		lr.Status = tdrpc.COMPLETED
	}

	// Update the status and the balance
	if plrerr := s.rpcStore.ProcessLedgerRecord(ctx, lr); plrerr != nil {
		return nil, grpc.Errorf(codes.Internal, "%v", plrerr)
	}

	// If there was an error, the ledger has been updated, return the error now
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Could not Pay: %v", err)
	}

	return &tdrpc.PayResponse{
		Result: lr,
	}, nil
}

// Ledger will return the ledger for a user
func (s *RPCServer) Ledger(ctx context.Context, request *tdrpc.LedgerRequest) (*tdrpc.LedgerResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, grpc.Errorf(codes.Internal, "Missing Account")
	}

	lrs, err := s.rpcStore.GetLedger(ctx, account.Id)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Error on GetLedger: %v", err)
	}

	return &tdrpc.LedgerResponse{
		Ledger: lrs,
	}, nil

}

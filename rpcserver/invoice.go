package rpcserver

import (
	"context"
	"encoding/hex"
	// "fmt"

	// "github.com/davecgh/go-spew/spew"
	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// DecodePayReq passes through decoding the pay request
func (s *RPCServer) DecodePayReq(ctx context.Context, request *lnrpc.PayReqString) (*lnrpc.PayReq, error) {

	// Decode and return the PayRequest
	return s.lclient.DecodePayReq(ctx, request)

}

// DecodePayReq passes through decoding the pay request
func (s *RPCServer) AddInvoice(ctx context.Context, request *tdrpc.AddInvoiceRequest) (*tdrpc.AddInvoiceResponse, error) {

	// Get the authenticated user from the context
	user := getUser(ctx)
	if user == nil {
		return nil, grpc.Errorf(codes.Internal, "Did not fetch user from context")
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

	// Store the userID <-> paymentHash association
	err = s.rpcStore.AddInvoice(ctx, user.Id, hex.EncodeToString(invoice.RHash))
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Could not AddInvoice: %v", err)
	}

	return &tdrpc.AddInvoiceResponse{
		PayReq: invoice.PaymentRequest,
	}, nil

}

// PayInvoice pays an invoice on behalf of the user
func (s *RPCServer) PayInvoice(ctx context.Context, request *tdrpc.PayInvoiceRequest) (*tdrpc.PayInvoiceResponse, error) {

	response, err := s.lclient.SendPaymentSync(ctx, &lnrpc.SendRequest{
		Amt:            request.Value,
		PaymentRequest: request.Invoice,
	})
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "Could not SendPaymentSync: %v", err)
	}

	return &tdrpc.PayInvoiceResponse{
		Error: response.GetPaymentError(),
	}, nil
}

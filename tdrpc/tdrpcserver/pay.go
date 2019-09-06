package tdrpcserver

import (
	"context"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/tdrpc"
	"git.coinninja.net/backend/thunderdome/thunderdome"
)

// Pay will pay a payment request
func (s *tdRPCServer) Pay(ctx context.Context, request *tdrpc.PayRequest) (*tdrpc.PayResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, status.Errorf(codes.Internal, "Missing Account")
	}

	if account.Locked {
		return nil, status.Errorf(codes.PermissionDenied, "Account is locked")
	}

	// Decode the Request
	pr, err := s.lclient.DecodePayReq(ctx, &lnrpc.PayReqString{PayReq: request.Request})
	if err != nil {
		return nil, err
	}

	// Check for expiration
	expiresAt := time.Unix(pr.Timestamp+pr.Expiry, 0)
	if time.Now().UTC().After(expiresAt) {
		return nil, status.Errorf(codes.InvalidArgument, "Request is expired")
	}

	// Check for mangled amount
	if pr.NumSatoshis < 0 || request.Value < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid value for payment request or payment value.")
	}

	// Check for zero amount
	if pr.NumSatoshis == 0 {
		if request.Value == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "Amount must be specified when paying a zero amount invoice")
		}
		// request.Value has the value we will pay already

		// The payment request has a value specified
	} else {
		// Ensure the user hasn't tried to specify a value, or if they have, it matches what the payment request is
		if request.Value != 0 && request.Value != pr.NumSatoshis {
			return nil, status.Errorf(codes.InvalidArgument, "You can only specify a value for a 0 sat invoice or the value must equal the invoice value of %d", pr.NumSatoshis)
		}
		// Force the request value to match the payment request
		request.Value = pr.NumSatoshis
	}

	// Build the ledger record
	lr := &tdrpc.LedgerRecord{
		Id:            pr.PaymentHash,
		AccountId:     account.Id,
		ExpiresAt:     &expiresAt,
		Status:        tdrpc.PENDING,
		Type:          tdrpc.LIGHTNING,
		Direction:     tdrpc.OUT,
		Value:         request.Value,
		ProcessingFee: int64((config.GetFloat64("tdome.processing_fee_rate") / 100.0) * float64(request.Value)),
		Memo:          pr.Description,
		Request:       request.Request,
	}

	// If it's not another user using this service, calcuate the network fee
	if pr.Destination != s.myPubKey {
		routesResponse, err := s.lclient.QueryRoutes(ctx, &lnrpc.QueryRoutesRequest{
			PubKey: pr.Destination,
			Amt:    lr.Value + lr.ProcessingFee,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "%v", err)
		} else if len(routesResponse.Routes) != 1 {
			return nil, status.Errorf(codes.Internal, "did not get network route")
		}
		lr.NetworkFee = routesResponse.Routes[0].TotalFees
	}

	// Sanity check the network fee
	if lr.NetworkFee > config.GetInt64("tdome.network_fee_limit") {
		return nil, status.Errorf(codes.Internal, "network fee too large: %d", lr.NetworkFee)
	}

	// If this is a payment to someone else using this service, mark the outbound records as interal
	if pr.Destination == s.myPubKey {
		lr.Id += thunderdome.InternalIdSuffix
	}

	// Save the initial state - will do some sanity checking as well
	err = s.store.ProcessLedgerRecord(ctx, lr)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// If this is a payment to someone else using this service, we transfer the balance internally
	if pr.Destination == s.myPubKey {

		// This is an internal payment, process the record
		lr, err = s.store.ProcessInternal(ctx, pr.PaymentHash)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
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

	// TODO: Determine if route taken was not the same as the quoted route and account for fee difference

	// Update the status and the balance
	if plrerr := s.store.ProcessLedgerRecord(ctx, lr); plrerr != nil {
		return nil, status.Errorf(codes.Internal, "%v", plrerr)
	}

	// If there was an error, the ledger has been updated, return the error now
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not Pay: %v", err)
	}

	return &tdrpc.PayResponse{
		Result: lr,
	}, nil
}

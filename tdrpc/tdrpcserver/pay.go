package tdrpcserver

import (
	"context"
	"strings"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// Pay will pay a payment request
func (s *tdRPCServer) Pay(ctx context.Context, request *tdrpc.PayRequest) (*tdrpc.LedgerRecordResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, tdrpc.ErrNotFound
	}

	if account.Locked {
		return nil, tdrpc.ErrAccountLocked
	}

	// If we're an agent, we are only allowed to proceed when we provide a PreAuthId
	if isAgent(ctx) && request.PreAuthId == "" {
		return nil, tdrpc.ErrPermissionDenied
	}

	// Decode the Request
	pr, err := s.lclient.DecodePayReq(ctx, &lnrpc.PayReqString{PayReq: request.Request})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not DecodePayReq: %v", status.Convert(err).Message())
	}

	// Check for expiration
	expiresAt := time.Unix(pr.Timestamp+pr.Expiry, 0)
	if time.Now().UTC().After(expiresAt) {
		return nil, tdrpc.ErrRequestExpired
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

	s.logger.Debugw("request.pay", "account_id", account.Id, zap.Any("request", lr))

	// Perform a quick sanity check to ensure we're not trying to pay ourself
	if pr.Destination == s.myPubKey {
		lrIn, err := s.store.GetLedgerRecord(ctx, lr.Id, tdrpc.IN)
		if err != nil {
			s.logger.Errorw("GetLedgerRecord Error", "id", lr.Id, "error", err)
			return nil, status.Errorf(codes.Internal, "GetLedgerRecord internal error")
		}
		// If the user it trying to pay themself, error out
		if lrIn.AccountId == lr.AccountId {
			return nil, tdrpc.ErrCannotPaySelfInvoice
		}

	}

	// If it's not another user using this service, calcuate the network fee
	if pr.Destination != s.myPubKey {
		queryRoutesRequest := &lnrpc.QueryRoutesRequest{
			PubKey: pr.Destination,
			Amt:    lr.Value + lr.ProcessingFee,
		}
		routesResponse, err := s.lclient.QueryRoutes(ctx, queryRoutesRequest)
		if err != nil {
			if strings.Contains(status.Convert(err).Message(), "unable to find a path") {
				return nil, tdrpc.ErrNoRouteFound
			}
			s.logger.Errorw("LND QueryRoutes Error", zap.Any("request", queryRoutesRequest), "error", err)
			return nil, status.Errorf(codes.Internal, "LND QueryRoutes internal error")
		} else if len(routesResponse.Routes) == 0 {
			return nil, tdrpc.ErrNoRouteFound
		}
		lr.NetworkFee = routesResponse.Routes[0].TotalFees
	}

	// Sanity check the network fee
	if lr.NetworkFee > config.GetInt64("tdome.network_fee_limit") {
		return nil, status.Errorf(codes.InvalidArgument, "Required network fee too large: %d", lr.NetworkFee)
	}

	// If this is a payment to someone else using this service, mark the outbound records as interal
	if pr.Destination == s.myPubKey {
		lr.Id += tdrpc.InternalIdSuffix
	}

	// If we're just providing an estimate, return it
	if request.Estimate {
		return &tdrpc.LedgerRecordResponse{
			Result: lr,
		}, nil
	}

	// Check if the request is pre-authorized and change the Id to match this request to update it
	if request.PreAuthId != "" {
		preAuthLr, err := s.store.GetLedgerRecord(ctx, request.PreAuthId, tdrpc.OUT)
		if err == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "Pre-Authorized payment not found")
		} else if err != nil {
			s.logger.Errorw("GetLedgerRecord Error", "preauth_id", request.PreAuthId, "error", err)
			return nil, status.Errorf(codes.Internal, "GetLedgerRecord internal error")
		} else if preAuthLr.Status != tdrpc.PENDING || preAuthLr.Request != tdrpc.PreAuthRequest {
			s.logger.Errorw("GetLedgerRecord Error", "preauth_id", request.PreAuthId, zap.Any("preauth_lr", preAuthLr), "error", err)
			return nil, status.Errorf(codes.Internal, "GetLedgerRecord internal error")
		}

		// We found the pre-authorizazed/reserved funds. Update the record to the current ID
		// This could return an error if the request was already paid
		err = s.store.UpdateLedgerRecordID(ctx, preAuthLr.Id, lr.Id, tdrpc.OUT)
		if err != nil {
			// Already exists means already paid
			if err == store.ErrAlreadyExists {
				return nil, tdrpc.ErrRequestAlreadyPaid
			}
			// A valid message is provided with this error
			if status.Code(err) == codes.InvalidArgument {
				return nil, err
			}
			s.logger.Errorw("UpdateLedgerRecordID Error", "prev", preAuthLr.Id, "next", lr.Id)
			return nil, status.Errorf(codes.Internal, "UpdateLedgerRecordID internal error")
		}
	}

	// Save the initial state - will do some sanity checking as well
	err = s.store.ProcessLedgerRecord(ctx, lr)
	if err != nil {
		// A valid message is provided with this error
		if status.Code(err) == codes.InvalidArgument {
			return nil, err
		}
		s.logger.Errorw("ProcessLedgerRecord Error", zap.Any("lr", lr), "error", err)
		return nil, status.Errorf(codes.Internal, "ProcessLedgerRecord internal error")
	}

	// ********* AT THIS POINT IN TIME ALL FUNCTIONS BELOW MUST COMPLETE *********
	// DO NOT ALLOW THE REQUEST CONTEXT TO CANCEL ANY OPERATION IN PROGRESS
	ctx = context.Background()

	// If this is a payment to someone else using this service, we transfer the balance internally
	if pr.Destination == s.myPubKey {

		// This is an internal payment, process the record
		intLr, err := s.store.ProcessInternal(ctx, pr.PaymentHash, lr)
		if err != nil {

			// Mark the original record as failed
			lr.Status = tdrpc.FAILED
			if prlErr := s.store.ProcessLedgerRecord(ctx, lr); prlErr != nil {
				s.logger.Errorw("ProcessLedgerRecord Error", zap.Any("lr", lr), "error", err)
			}

			// A valid message is provided with this error
			if status.Code(err) == codes.InvalidArgument {
				return nil, err
			}
			s.logger.Errorw("ProcessInternal Error", zap.Any("lr", lr), "error", err)
			return nil, status.Errorf(codes.Internal, "ProcessInternal error")
		}

		return &tdrpc.LedgerRecordResponse{
			Result: intLr,
		}, nil

	}

	// Send the payment
	sendPaymentSyncRequest := &lnrpc.SendRequest{
		Amt:            request.Value,
		PaymentRequest: request.Request,
	}
	response, err := s.lclient.SendPaymentSync(ctx, sendPaymentSyncRequest)
	if err != nil {
		// The payment is still in transition, it could end up getting paid. Leave it for now as pending.
		if strings.Contains(err.Error(), "transition") { // Error should be: payment is in transition
			lr.Status = tdrpc.PENDING
			lr.Error = err.Error()
		} else {
			lr.Status = tdrpc.FAILED
			lr.Error = err.Error()
		}
	} else if response.PaymentError != "" {
		lr.Status = tdrpc.FAILED
		lr.Error = response.PaymentError
	} else {
		lr.Status = tdrpc.COMPLETED
	}

	// TODO: Determine if route taken was not the same as the quoted route and account for fee difference

	// Update the status and the balance - Ensure it completes outside of this request context
	if plrerr := s.store.ProcessLedgerRecord(ctx, lr); plrerr != nil {
		// A valid message is provided with this error
		if status.Code(plrerr) == codes.InvalidArgument {
			return nil, plrerr
		}
		s.logger.Errorw("ProcessLedgerRecord Error", zap.Any("lr", lr), "error", err)
		return nil, status.Errorf(codes.Internal, "ProcessLedgerRecord internal error")
	}

	// If there was an error, the ledger has been updated, return the error now
	if err != nil {
		s.logger.Errorw("LND SendPaymentSync Error", zap.Any("request", sendPaymentSyncRequest), "error", err)
		return nil, status.Errorf(codes.Internal, "Could not SendPaymentSync: %v", status.Convert(err).Message())
	}

	return &tdrpc.LedgerRecordResponse{
		Result: lr,
	}, nil
}

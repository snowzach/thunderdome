package tdrpcserver

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
	"go.uber.org/zap"
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
		return nil, tdrpc.ErrNotFound
	}

	if account.Locked {
		return nil, tdrpc.ErrAccountLocked
	}

	if request.Value < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid Value")
	} else if request.Value > config.GetInt64("tdome.value_limit") {
		return nil, status.Errorf(codes.InvalidArgument, "Max invoice value is %s sats", tdrpc.FormatInt(ctx, config.GetInt64("tdome.value_limit")))
	}

	if request.Expires != 0 && (request.Expires < 300 || request.Expires > 7776000) {
		return nil, status.Errorf(codes.InvalidArgument, "Expires cannot be less than %s or greater than %s seconds", tdrpc.FormatInt(ctx, 300), tdrpc.FormatInt(ctx, 7776000))
	}
	if request.Expires == 0 {
		request.Expires = config.GetInt64("tdome.default_request_expires")
	}

	// Get pending incoming bitcoin balance for this user
	pendingStats, err := s.store.GetLedgerRecordStats(ctx, map[string]string{
		"account_id": account.Id,
		"type":       tdrpc.LIGHTNING.String(),
		"direction":  tdrpc.IN.String(),
		"status":     tdrpc.PENDING.String(),
		"generated":  "false",
	}, time.Time{})
	if err != nil {
		s.logger.Errorw("GetLedgerRecordStats Error", "error", err)
		return nil, status.Errorf(codes.Internal, "GetLedgerRecordStats internal error")
	}

	if pendingStats.Count >= config.GetInt64("tdome.create_request_limit") {
		return nil, tdrpc.ErrCreateRequestLimitExceeded
	}

	// Create the invoice
	addInvoiceRequest := &lnrpc.Invoice{
		Memo:   request.Memo,
		Value:  request.Value,
		Expiry: request.Expires,
	}
	invoice, err := s.lclient.AddInvoice(ctx, addInvoiceRequest)
	if err != nil {
		s.logger.Errorw("LND AddInvoice Error", zap.Any("request", addInvoiceRequest), "error", err)
		return nil, status.Errorf(codes.Internal, "Could not AddInvoice: %s", status.Convert(err).Message())
	}

	// Get the expires time
	expiresAt := time.Now().UTC().Add(time.Duration(request.Expires) * time.Second)

	// Put it in the ledger
	lr := &tdrpc.LedgerRecord{
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
	}

	s.logger.Debugw("request.create", "account_id", account.Id, zap.Any("request", lr))

	err = s.store.ProcessLedgerRecord(ctx, lr)
	if err != nil {
		// A valid message is provided with this error
		if status.Code(err) == codes.InvalidArgument {
			return nil, err
		}
		s.logger.Errorw("ProcessLedgerRecord Error", zap.Any("lr", lr), "error", err)
		return nil, status.Errorf(codes.Internal, "ProcessLedgerRecord internal error")
	}

	// Return the payment request
	return &tdrpc.CreateResponse{
		Request: invoice.PaymentRequest,
	}, nil

}

// CreateGenerated makes a payment request with no value. If one exists already, it will be returned.
func (s *tdRPCServer) CreateGenerated(ctx context.Context, request *tdrpc.CreateGeneratedRequest) (*tdrpc.CreateResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, tdrpc.ErrNotFound
	}

	// If it's locked
	if account.Locked {
		// If we're not the agent, access denied
		if !isAgent(ctx) {
			return nil, tdrpc.ErrAccountLocked
		}
		// If we don't specifically allow locked accounts, return not found
		if !request.AllowLocked {
			return nil, tdrpc.ErrNotFound
		}
	}

	// See if we already have an existing invoice
	lr, err := s.store.GetActiveGeneratedLightningLedgerRequest(ctx, account.Id)
	if err == nil {
		// Found one, return it
		return &tdrpc.CreateResponse{
			Request: lr.Request,
		}, nil
	} else if err != store.ErrNotFound {
		// Some other error
		s.logger.Errorw("GetActiveGeneratedLightningLedgerRequest Error", "account_id", account.Id, "error", err)
		return nil, status.Errorf(codes.Internal, "GetActiveGeneratedLightningLedgerRequest internal error")
	}

	expirationSeconds := config.GetInt64("tdome.create_generated_expires")
	expiresAt := time.Now().UTC().Add(time.Duration(expirationSeconds) * time.Second)

	// Create the invoice
	addInvoiceRequest := &lnrpc.Invoice{
		Value:  0,
		Expiry: expirationSeconds,
	}
	invoice, err := s.lclient.AddInvoice(ctx, addInvoiceRequest)
	if err != nil {
		s.logger.Errorw("LND AddInvoice Error", zap.Any("request", addInvoiceRequest), "error", err)
		return nil, status.Errorf(codes.Internal, "Could not AddInvoice: %v", status.Convert(err).Message())
	}

	// Create a new one and put it in the ledger
	lr = &tdrpc.LedgerRecord{
		Id:        hex.EncodeToString(invoice.RHash),
		AccountId: account.Id,
		ExpiresAt: &expiresAt,
		Status:    tdrpc.PENDING,
		Generated: true,
		Hidden:    true, // Initially mark as hidden
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.IN,
		Value:     0,
		AddIndex:  invoice.AddIndex,
		Request:   invoice.PaymentRequest,
	}
	err = s.store.ProcessLedgerRecord(ctx, lr)
	if err != nil {
		// A valid message is provided with this error
		if status.Code(err) == codes.InvalidArgument {
			return nil, err
		}
		s.logger.Errorw("ProcessLedgerRecord Error", zap.Any("lr", lr), "error", err)
		return nil, status.Errorf(codes.Internal, "ProcessLedgerRecord internal error")
	}

	// Return the payment request
	return &tdrpc.CreateResponse{
		Request: invoice.PaymentRequest,
	}, nil

}

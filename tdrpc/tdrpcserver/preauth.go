package tdrpcserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	config "github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// CreatePreAuth will will preauthorize a payment to be made on the users behalf
func (s *tdRPCServer) CreatePreAuth(ctx context.Context, request *tdrpc.CreateRequest) (*tdrpc.LedgerRecordResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, tdrpc.ErrNotFound
	}

	if account.Locked {
		return nil, tdrpc.ErrAccountLocked
	}

	if request.Value <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid Value")
	}

	if request.Expires != 0 && (request.Expires < 300 || request.Expires > 7776000) {
		return nil, status.Errorf(codes.InvalidArgument, "Expires cannot be less than %s or greater than %s seconds", tdrpc.FormatInt(ctx, 300), tdrpc.FormatInt(ctx, 7776000))
	}

	// If we're an agent, we can only request a pre-auth below the tdome.agent_pay_value_limit
	if isAgent(ctx) && request.Value > config.GetInt64("tdome.agent_pay_value_limit") {
		return nil, tdrpc.ErrPermissionDenied
	}

	if request.Expires == 0 {
		request.Expires = config.GetInt64("tdome.default_request_expires")
	}

	// Generate a random hex string to use as a temporary identifier to reserve funds
	randomID := make([]byte, 32)
	if _, err := rand.Read(randomID); err != nil {
		return nil, status.Errorf(codes.Internal, "could not get random id")
	}
	tempLedgerRecordID := tdrpc.PreAuthLedgerRecordIdPrefix + hex.EncodeToString(randomID)

	// Get the expires time
	expiresAt := time.Now().UTC().Add(time.Duration(request.Expires) * time.Second)

	// Put it in the ledger
	lr := &tdrpc.LedgerRecord{
		Id:        tempLedgerRecordID,
		AccountId: account.Id,
		ExpiresAt: &expiresAt,
		Status:    tdrpc.PENDING,
		Type:      tdrpc.LIGHTNING,
		Direction: tdrpc.OUT,
		Value:     request.Value,
		Memo:      request.Memo,
		Request:   tdrpc.PreAuthRequest,
	}

	s.logger.Debugw("request.preauth", zap.Any("lr", lr))

	// Save the initial state - will do some sanity checking as well
	err := s.store.ProcessLedgerRecord(ctx, lr)
	if err != nil {
		// A valid message is provided with this error
		if status.Code(err) == codes.InvalidArgument {
			return nil, err
		}
		s.logger.Errorw("ProcessLedgerRecord Error", zap.Any("lr", lr), "error", err)
		return nil, status.Errorf(codes.Internal, "ProcessLedgerRecord internal error")
	}

	return &tdrpc.LedgerRecordResponse{
		Result: lr,
	}, nil

}

// GetPreAuth will get a pre-auth record
func (s *tdRPCServer) GetPreAuth(ctx context.Context, request *tdrpc.Id) (*tdrpc.LedgerRecordResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, tdrpc.ErrNotFound
	}

	if account.Locked {
		return nil, tdrpc.ErrAccountLocked
	}

	// Ensure the request has the PreAuth prefix
	if !strings.HasPrefix(request.Id, tdrpc.PreAuthLedgerRecordIdPrefix) {
		return nil, tdrpc.ErrNotFound
	}

	lr, err := s.store.GetLedgerRecord(ctx, request.Id, tdrpc.OUT)
	if err == store.ErrNotFound {
		return nil, tdrpc.ErrNotFound
	} else if err != nil {
		// A valid message is provided with this error
		if status.Code(err) == codes.InvalidArgument {
			return nil, err
		}
		s.logger.Errorw("GetLedgerRecord Error", zap.Any("request", request), "error", err)
		return nil, status.Errorf(codes.Internal, "GetLedgerRecord internal error")
	}

	return &tdrpc.LedgerRecordResponse{
		Result: lr,
	}, nil

}

// ExpirePreAuth will expire a pre-authorization request returning the funds to the user
func (s *tdRPCServer) ExpirePreAuth(ctx context.Context, request *tdrpc.Id) (*tdrpc.LedgerRecordResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, tdrpc.ErrNotFound
	}

	if account.Locked {
		return nil, tdrpc.ErrAccountLocked
	}

	lr, err := s.store.GetLedgerRecord(ctx, request.Id, tdrpc.OUT)
	if err == store.ErrNotFound {
		return nil, tdrpc.ErrNotFound
	} else if err != nil {
		// A valid message is provided with this error
		if status.Code(err) == codes.InvalidArgument {
			return nil, err
		}
		s.logger.Errorw("GetLedgerRecord Error", zap.Any("request", request), "error", err)
		return nil, status.Errorf(codes.Internal, "GetLedgerRecord internal error")
	}

	// If the account id doesn't match, deny access
	if lr.AccountId != account.Id {
		return nil, tdrpc.ErrNotFound
	}

	// We only support making the update when the status is pending and it was a PreAuth request
	if lr.Status == tdrpc.PENDING && lr.Request == tdrpc.PreAuthRequest {

		lr.Status = tdrpc.EXPIRED

		// Save the update
		err := s.store.ProcessLedgerRecord(ctx, lr)
		if err != nil {
			// A valid message is provided with this error
			if status.Code(err) == codes.InvalidArgument {
				return nil, err
			}
			s.logger.Errorw("ProcessLedgerRecord Error", zap.Any("lr", lr), "error", err)
			return nil, status.Errorf(codes.Internal, "ProcessLedgerRecord internal error")
		}

	}

	return &tdrpc.LedgerRecordResponse{
		Result: lr,
	}, nil

}

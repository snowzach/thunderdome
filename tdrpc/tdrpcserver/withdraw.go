package tdrpcserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// Pay will pay a payment request
func (s *tdRPCServer) Withdraw(ctx context.Context, request *tdrpc.WithdrawRequest) (*tdrpc.WithdrawResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, tdrpc.ErrNotFound
	}

	if account.Locked {
		return nil, tdrpc.ErrAccountLocked
	}

	// What we charge to withdraw (percentage)
	withdrawFeeRate := config.GetFloat64("tdome.withdraw_fee_rate") / 100.0

	// Get pending incoming bitcoin balance for this user
	pendingStats, err := s.store.GetLedgerRecordStats(ctx, map[string]string{
		"account_id": account.Id,
		"type":       tdrpc.BTC.String(),
		"direction":  tdrpc.IN.String(),
		"status":     tdrpc.COMPLETED.String(),
		"request":    tdrpc.RequestInstantPending,
	}, time.Time{})
	if err != nil {
		s.logger.Errorw("GetLedgerRecordStats Error", "error", err)
		return nil, status.Errorf(codes.Internal, "GetLedgerRecordStats internal error")
	}

	// Check if this is an account sweep give them all the confirmed funds
	if request.Value == tdrpc.ValueSweep {
		request.Value = account.Balance - pendingStats.Value
		// Ensure there is enough to allow a withdraw still
		if request.Value < config.GetInt64("tdome.withdraw_min") {
			if pendingStats.Value > 0 {
				return nil, status.Errorf(codes.InvalidArgument, "The confirmed balance of %s sats is too small for this withdraw to pay network fees. You have %s sats still pending confirmation.", tdrpc.FormatInt(ctx, account.Balance-pendingStats.Value), tdrpc.FormatInt(ctx, pendingStats.Value))
			} else {
				return nil, status.Errorf(codes.InvalidArgument, "The balance of %s sats is too small for this withdraw to pay network fees.", tdrpc.FormatInt(ctx, request.Value))
			}
		}
	}

	// If there is a pending value, ensure there is sufficient confirmed value to withdraw the requested amount
	if request.Value > account.Balance-pendingStats.Value {
		return nil, status.Errorf(codes.InvalidArgument, "The confirmed balance of %s sats is insufficient for this withdraw. You have %s sats still pending confirmation.", tdrpc.FormatInt(ctx, account.Balance-pendingStats.Value), tdrpc.FormatInt(ctx, pendingStats.Value))
	}

	if request.Value < config.GetInt64("tdome.withdraw_min") {
		return nil, status.Errorf(codes.InvalidArgument, "Withdraw value must be at least %s satoshis", tdrpc.FormatInt(ctx, config.GetInt64("tdome.withdraw_min")))
	}

	// The adjustedValue will be the target amount we wish to withdraw after taking processing and network fees
	// Estimate the target value based on an estimated fee of tdome.withdraw_fee_estimate sats and the withdraw fee rate
	// We need this to determine the SatPerByte below
	adjustedValue := int64(float64(request.Value-config.GetInt64("tdome.withdraw_fee_estimate")) / (1.0 + withdrawFeeRate))

	// Check if we specified blocks or fee rate
	if request.Blocks == 0 {
		// If no fee was specified
		if request.SatPerByte == 0 {
			request.Blocks = config.GetInt32("tdome.default_withdraw_target_blocks")
		} else {
			if request.SatPerByte > config.GetInt64("tdome.network_fee_limit") {
				return nil, status.Errorf(codes.InvalidArgument, "Fee rate must be less than %s sats/byte", tdrpc.FormatInt(ctx, config.GetInt64("tdome.network_fee_limit")))
			}
			// We're going to use an estimator of 6 blocks to determine the transaction size
			// We can then use our fee rate to determine how much to charge the user
			request.Blocks = 6
		}
		// Blocks != 0, if SatPerByte also != 0, error
	} else if request.SatPerByte != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "You must specify blocks or sat_per_byte but not both")
	} else if request.Blocks < 1 || request.Blocks > 144 {
		return nil, status.Errorf(codes.InvalidArgument, "Blocks value must be between 1-144")
	}

	// Get the fee required based on target blocks - we need to do this regardless if blocs or sats_per_byte so we know the estimated transaction size
	estimateFeeRequest := &lnrpc.EstimateFeeRequest{
		AddrToAmount: map[string]int64{request.Address: adjustedValue},
		TargetConf:   request.Blocks,
	}
	feeResponse, err := s.lclient.EstimateFee(ctx, estimateFeeRequest)
	if err != nil {
		s.logger.Errorw("LND EstimateFee Error", zap.Any("request", estimateFeeRequest), "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "Could not EstimateFee: %v", status.Convert(err).Message())
	}

	// By default we use the fee from the Sat
	networkFee := feeResponse.FeeSat
	if request.SatPerByte == 0 {
		// Get the SatPerByte to use for sending the actual transaction
		request.SatPerByte = feeResponse.FeerateSatPerByte
	} else {
		// We specified what SatPerByte we want to use, get the txSize and figure out the networkFee base on calculated txSize
		txSize := int64(float64(feeResponse.FeeSat) / float64(feeResponse.FeerateSatPerByte))
		networkFee = txSize * request.SatPerByte
	}

	// Modify the adjustedValue to whatever the actual value needs to be to hit the request.Value
	adjustedValue = int64(float64(request.Value-networkFee) / (1.0 + withdrawFeeRate))
	processingFee := int64(float64(adjustedValue) * withdrawFeeRate)

	// Handle any rounding errors, update the request.Value
	request.Value = request.Value - networkFee - processingFee

	// Make sure the fees didn't eat up any possible withdraw
	if request.Value <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "The account balance is too small to pay the transactions fees requested")
	}

	// Generate a random hex string to use as a temporary identifier to reserve funds
	randomID := make([]byte, 32)
	if _, err := rand.Read(randomID); err != nil {
		return nil, status.Errorf(codes.Internal, "could not get random id")
	}
	tempLedgerRecordID := tdrpc.TempLedgerRecordIdPrefix + hex.EncodeToString(randomID)

	// Build the temporary ledger record
	lr := &tdrpc.LedgerRecord{
		Id:            tempLedgerRecordID,
		AccountId:     account.Id,
		Status:        tdrpc.PENDING,
		Type:          tdrpc.BTC,
		Direction:     tdrpc.OUT,
		Value:         request.Value,
		NetworkFee:    networkFee,
		ProcessingFee: processingFee,
		Memo:          fmt.Sprintf("Withdraw %d sats with %d sat netowrk fee and %d sat processing fee to %s", request.Value, networkFee, processingFee, request.Address),
	}

	s.logger.Debugw("request.withdraw", "account_id", account.Id, zap.Any("request", lr))

	// If we are just estimating, return the result without processing
	if request.Estimate {
		var t = time.Now()
		lr.CreatedAt = &t
		lr.UpdatedAt = &t
		lr.Id = "estimate"
		return &tdrpc.WithdrawResponse{
			Result: lr,
		}, nil
	}

	// Save the initial state - will do some sanity checking as well and preallocate funds
	err = s.store.ProcessLedgerRecord(ctx, lr)
	if err != nil {
		// A valid message is provided with this error
		if status.Code(err) == codes.InvalidArgument {
			return nil, err
		}
		s.logger.Errorw("ProcessLedgerRecord Error", zap.Any("lr", lr), "error", err)
		return nil, status.Errorf(codes.Internal, "ProcessLedgerRecord internal error")
	}

	sendCoinsRequest := &lnrpc.SendCoinsRequest{
		Addr:       request.Address,
		Amount:     request.Value,
		SatPerByte: request.SatPerByte,
	}

	// Send the payment
	response, err := s.lclient.SendCoins(ctx, sendCoinsRequest)
	if err != nil {
		lr.Status = tdrpc.FAILED
		lr.Error = err.Error()

		// Update the record to failed - return funds
		if plrerr := s.store.ProcessLedgerRecord(ctx, lr); plrerr != nil {
			// A valid message is provided with this error
			if status.Code(plrerr) == codes.InvalidArgument {
				return nil, plrerr
			}
			s.logger.Errorw("ProcessLedgerRecord Error", zap.Any("lr", lr), "error", err)
			return nil, status.Errorf(codes.Internal, "ProcessLedgerRecord internal error")
		}
	}

	// If there was an error, the ledger has been updated, return the error now
	if err != nil {
		s.logger.Errorw("LND SendCoins Error", zap.Any("request", sendCoinsRequest), "error", err)
		return nil, status.Errorf(codes.Internal, "Could not SendCoins: %v", status.Convert(err).Message())
	}

	// Otherwise we succeeded, update the ledger record ID to be the transaction id
	err = s.store.UpdateLedgerRecordID(ctx, tempLedgerRecordID, response.Txid, tdrpc.OUT)
	if err != nil {
		// A valid message is provided with this error
		if status.Code(err) == codes.InvalidArgument {
			return nil, err
		}
		s.logger.Errorw("UpdateLedgerRecordID Error", "prev", tempLedgerRecordID, "next", response.Txid)
		return nil, status.Errorf(codes.Internal, "UpdateLedgerRecordID internal error")
	}

	lr.Id = response.Txid

	return &tdrpc.WithdrawResponse{
		Result: lr,
	}, nil
}

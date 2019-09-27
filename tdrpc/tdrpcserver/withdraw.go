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
		return nil, status.Errorf(codes.Internal, "Missing Account")
	}

	if account.Locked {
		return nil, status.Errorf(codes.PermissionDenied, "Account is locked")
	}

	// What we charge to withdraw (percentage)
	withdrawFeeRate := config.GetFloat64("tdome.withdraw_fee_rate") / 100.0

	// Are we sweeping the account
	var accountSweep = false
	if request.Value == tdrpc.ValueSweep {
		accountSweep = true
		// Estimate the base value based on an estimated fee of 2000 sats and the withdraw fee rate
		// We need this to determine the SatPerByte below
		request.Value = int64(float64(account.Balance-2000) / (1.0 + withdrawFeeRate))

		// Check for mangled amount
	} else if request.Value < config.GetInt64("tdome.min_withdraw") {
		return nil, status.Errorf(codes.InvalidArgument, "Withdraw value must be at least %d satoshis", config.GetInt64("tdome.min_withdraw"))
	}

	// Check if we specified blocks or fee rate
	if request.Blocks == 0 {
		// If no fee was specified
		if request.SatPerByte == 0 {
			request.Blocks = config.GetInt32("tdome.default_withdraw_target_blocks")
		} else {
			if request.SatPerByte > config.GetInt64("tdome.network_fee_limit") {
				return nil, status.Errorf(codes.InvalidArgument, "Fee rate must be less than %d sats/byte", config.GetInt64("tdome.network_fee_limit"))
			}
			// We're going to use an estimator of 6 blocks to determine the transaction size
			// We can then use our fee rate to determine how much to charge the user
			request.Blocks = 6
		}
		// Blocks != 0, if SatPerByte also != 0, error
	} else if request.SatPerByte != 0 {
		return nil, status.Errorf(codes.InvalidArgument, "You must specify blocks or sat_per_byte but not both")
	} else if request.Blocks < 0 || request.Blocks > 144 {
		return nil, status.Errorf(codes.InvalidArgument, "Blocks value must be between 0-%d", config.GetInt64("tdome.min_withdraw"))
	}

	// Get the fee required based on target blocks
	feeResponse, err := s.lclient.EstimateFee(ctx, &lnrpc.EstimateFeeRequest{
		AddrToAmount: map[string]int64{request.Address: request.Value},
		TargetConf:   request.Blocks,
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not EstimateFee: %v", err)
	}

	// By default we use the fee from the Sat
	networkFee := feeResponse.FeeSat
	if request.SatPerByte == 0 {
		// Get the SatPerByte to use for sending the actual transaction
		request.SatPerByte = feeResponse.FeerateSatPerByte
	} else {
		// We specified what SatPerByte we want to use, get the txSize and figure out the networkFee
		txSize := int64(float64(feeResponse.FeeSat) / float64(feeResponse.FeerateSatPerByte))
		networkFee = txSize * request.SatPerByte
	}

	// Calculate the processing fee
	processingFee := int64(withdrawFeeRate * float64(request.Value))

	// If this is an account sweep, we now know the exact networkFee at this point, update all the values to ensure the account is empty
	if accountSweep {
		processingFee = int64(float64(account.Balance-networkFee) * withdrawFeeRate)
		request.Value = account.Balance - networkFee - processingFee
	}

	// Generate a random hex string to use as a temporary identifier to reserve funds
	randomID := make([]byte, 32)
	if _, err := rand.Read(randomID); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not generate random id")
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
		Memo:          fmt.Sprintf("Withdraw %d sats with %d sat netowrk fee and %d sat processing fee", request.Value, feeResponse.FeeSat, processingFee),
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
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	// Send the payment
	response, err := s.lclient.SendCoins(ctx, &lnrpc.SendCoinsRequest{
		Addr:       request.Address,
		Amount:     request.Value,
		SatPerByte: request.SatPerByte,
	})
	if err != nil {
		lr.Status = tdrpc.FAILED
		lr.Error = err.Error()

		// Update the record to failed - return funds
		if plrerr := s.store.ProcessLedgerRecord(ctx, lr); plrerr != nil {
			return nil, status.Errorf(codes.Internal, "%v", plrerr)
		}
	}

	// If there was an error, the ledger has been updated, return the error now
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not Withdraw: %v", err)
	}

	// Otherwise we succeeded, update the ledger record ID to be the transaction id
	err = s.store.UpdateLedgerRecordID(ctx, tempLedgerRecordID, response.Txid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not UpdateLedgerRecordID: %v", err)
	}

	lr.Id = response.Txid

	return &tdrpc.WithdrawResponse{
		Result: lr,
	}, nil
}

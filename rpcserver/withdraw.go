package rpcserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/tdrpc"
	"git.coinninja.net/backend/thunderdome/thunderdome"
)

// Pay will pay a payment request
func (s *RPCServer) Withdraw(ctx context.Context, request *tdrpc.WithdrawRequest) (*tdrpc.WithdrawResponse, error) {

	// Get the authenticated user from the context
	account := getAccount(ctx)
	if account == nil {
		return nil, status.Errorf(codes.Internal, "Missing Account")
	}

	// Check for mangled amount
	if request.Value < config.GetInt64("tdome.min_withdraw") {
		return nil, status.Errorf(codes.InvalidArgument, "Withdraw value must be at least %d satoshis", config.GetInt64("tdome.min_withdraw"))
	}

	// Check the blocks value
	if request.Blocks < 0 || request.Blocks > 144 {
		return nil, status.Errorf(codes.InvalidArgument, "Blocks value must be between 0-%d", config.GetInt64("tdome.min_withdraw"))
	}

	// If zero, set default blocks
	if request.Blocks == 0 {
		request.Blocks = config.GetInt32("tdome.default_withdraw_target_blocks")
	}

	// Get the fee required
	feeResponse, err := s.lclient.EstimateFee(ctx, &lnrpc.EstimateFeeRequest{
		AddrToAmount: map[string]int64{request.Address: request.Value},
		TargetConf:   request.Blocks,
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not EstimateFee: %v", err)
	}

	// Generate a random hex string to use as a temporary identifier to reserve funds
	randomID := make([]byte, 32)
	if _, err := rand.Read(randomID); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not generate random id")
	}
	tempLedgerRecordID := thunderdome.TempLedgerRecordIdPrefix + hex.EncodeToString(randomID)

	// Build the temporary ledger record
	lr := &tdrpc.LedgerRecord{
		Id:            tempLedgerRecordID,
		AccountId:     account.Id,
		Status:        tdrpc.PENDING,
		Type:          tdrpc.BTC,
		Direction:     tdrpc.OUT,
		Value:         request.Value,
		NetworkFee:    feeResponse.FeeSat,
		ProcessingFee: int64((config.GetFloat64("tdome.withdraw_fee_rate") / 100.0) * float64(request.Value)),
		Memo:          fmt.Sprintf("Withdraw %d sats with %d sats fee", request.Value, feeResponse.FeeSat),
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
		SatPerByte: feeResponse.FeerateSatPerByte,
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

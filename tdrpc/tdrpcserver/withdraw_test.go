package tdrpcserver

import (
	"context"
	"testing"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	_ "git.coinninja.net/backend/thunderdome/conf"
	"git.coinninja.net/backend/thunderdome/mocks"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func TestWithdraw(t *testing.T) {

	// Mocks
	mockStore := new(mocks.Store)
	mockLClient := new(mocks.LightningClient)
	mockLClient.On("GetInfo", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*lnrpc.GetInfoRequest")).Once().Return(&lnrpc.GetInfoResponse{IdentityPubkey: "testing"}, nil)
	mockDCache := new(mocks.DistCache)

	// RPC Server
	s, err := NewTDRPCServer(mockStore, mockLClient, mockDCache)
	assert.Nil(t, err)

	// Bootstrap authentication
	account := &tdrpc.Account{
		Id:      "123123123123132132132123131123123123123132132132123131123123123333",
		Address: "2MsoezssHTCZbeoVcZ5NgYmtNiUpyzAc5hm",
		Balance: 100000,
	}
	ctx := addAccount(context.Background(), account)

	mockStore.On("GetLedgerRecordStats", mock.AnythingOfType("*context.valueCtx"), map[string]string{
		"account_id": account.Id,
		"type":       tdrpc.BTC.String(),
		"direction":  tdrpc.IN.String(),
		"status":     tdrpc.COMPLETED.String(),
		"request":    tdrpc.RequestInstantPending,
	}, mock.AnythingOfType("time.Time")).Once().Return(&tdrpc.LedgerRecordStats{Count: 0, Value: 0, NetworkFee: 0, ProcessingFee: 0}, nil)

	// To small amount
	_, err = s.Withdraw(ctx, &tdrpc.WithdrawRequest{
		Address: account.Address,
		Value:   10,
	})
	assert.NotNil(t, err)

	// AddInvoice call
	mockLClient.On("EstimateFee", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*lnrpc.EstimateFeeRequest")).Once().Return(
		&lnrpc.EstimateFeeResponse{
			FeeSat:            123,
			FeerateSatPerByte: 12,
		}, nil,
	)
	mockLClient.On("SendCoins", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*lnrpc.SendCoinsRequest")).Once().Return(
		&lnrpc.SendCoinsResponse{
			Txid: "abc1234",
		}, nil,
	)

	mockStore.On("GetLedgerRecordStats", mock.AnythingOfType("*context.valueCtx"), map[string]string{
		"account_id": account.Id,
		"type":       tdrpc.BTC.String(),
		"direction":  tdrpc.IN.String(),
		"status":     tdrpc.COMPLETED.String(),
		"request":    tdrpc.RequestInstantPending,
	}, mock.AnythingOfType("time.Time")).Once().Return(&tdrpc.LedgerRecordStats{}, nil)
	mockStore.On("ProcessLedgerRecord", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*tdrpc.LedgerRecord")).Once().Return(nil)
	mockStore.On("UpdateLedgerRecordID", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), tdrpc.OUT).Once().Return(nil)

	// Make the request
	_, err = s.Withdraw(ctx, &tdrpc.WithdrawRequest{
		Address: account.Address,
		Value:   50000,
	})
	assert.Nil(t, err)

	mockStore.AssertExpectations(t)
	mockLClient.AssertExpectations(t)

}

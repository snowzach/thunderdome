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

func TestCreatePreAuth(t *testing.T) {

	// Mocks
	mockStore := new(mocks.Store)
	mockLClient := new(mocks.LightningClient)
	mockLClient.On("GetInfo", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*lnrpc.GetInfoRequest")).Once().Return(&lnrpc.GetInfoResponse{IdentityPubkey: "testing"}, nil)

	// RPC Server
	s, err := NewTDRPCServer(mockStore, mockLClient)
	assert.Nil(t, err)

	// Bootstrap authentication
	account := &tdrpc.Account{
		Id:      "123123123123132132132123131123123123123132132132123131123123123333",
		Address: "2MsoezssHTCZbeoVcZ5NgYmtNiUpyzAc5hm",
		Balance: 10,
	}
	ctx := addAccount(context.Background(), account)

	mockStore.On("ProcessLedgerRecord", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*tdrpc.LedgerRecord")).Once().Return(tdrpc.ErrInsufficientFunds)

	// Insufficient funds
	_, err = s.CreatePreAuth(ctx, &tdrpc.CreateRequest{
		Memo:    "test",
		Value:   20,
		Expires: 600,
	})
	assert.NotNil(t, err)

	mockStore.On("ProcessLedgerRecord", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*tdrpc.LedgerRecord")).Once().Return(nil)

	// Insufficient funds
	_, err = s.CreatePreAuth(ctx, &tdrpc.CreateRequest{
		Memo:    "test",
		Value:   50,
		Expires: 600,
	})
	assert.Nil(t, err)

	mockStore.AssertExpectations(t)

}

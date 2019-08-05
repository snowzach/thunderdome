package rpcserver

import (
	"context"
	"testing"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"git.coinninja.net/backend/thunderdome/mocks"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func TestLedger(t *testing.T) {

	// Mocks
	mockStore := new(mocks.Store)
	mockLClient := new(mocks.LightningClient)
	mockLClient.On("GetInfo", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*lnrpc.GetInfoRequest")).Once().Return(&lnrpc.GetInfoResponse{IdentityPubkey: "testing"}, nil)
	// RPC Server
	s, err := NewRPCServer(mockStore, mockLClient)
	assert.Nil(t, err)

	// Bootstrap authentication
	account := &tdrpc.Account{
		Id:      "123123123123132132132123131123123123123132132132123131123123123333",
		Address: "2MsoezssHTCZbeoVcZ5NgYmtNiUpyzAc5hm",
	}
	ctx := addAccount(context.Background(), account)

	lrs := []*tdrpc.LedgerRecord{
		&tdrpc.LedgerRecord{
			Id: "1",
		},
		&tdrpc.LedgerRecord{
			Id: "2",
		},
	}

	mockStore.On("GetLedger", mock.AnythingOfType("*context.valueCtx"), account.Id).Once().Return(lrs, nil)

	// Make the request
	response, err := s.Ledger(ctx, &tdrpc.LedgerRequest{})
	assert.Nil(t, err)
	assert.Equal(t, &tdrpc.LedgerResponse{Ledger: lrs}, response)

	mockStore.AssertExpectations(t)
	mockLClient.AssertExpectations(t)

}

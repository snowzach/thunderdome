package tdrpcserver

import (
	"context"
	"testing"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"git.coinninja.net/backend/thunderdome/mocks"
	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func TestCreate(t *testing.T) {

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
	}
	ctx := addAccount(context.Background(), account)

	// Bad Value
	_, err = s.Create(ctx, &tdrpc.CreateRequest{
		Memo:  "test",
		Value: -1,
	})
	assert.NotNil(t, err)

	_, err = s.Create(ctx, &tdrpc.CreateRequest{
		Memo:    "test",
		Value:   1,
		Expires: -1,
	})
	assert.NotNil(t, err)

	// AddInvoice call
	mockLClient.On("AddInvoice", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*lnrpc.Invoice")).Once().Return(
		&lnrpc.AddInvoiceResponse{
			RHash:          []byte("asdfasdfasdf"),
			PaymentRequest: "whutwhute",
		}, nil,
	)
	mockStore.On("ProcessLedgerRecord", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*tdrpc.LedgerRecord")).Once().Return(nil)

	// Make the request
	_, err = s.Create(ctx, &tdrpc.CreateRequest{
		Memo:  "test",
		Value: 0,
	})
	assert.Nil(t, err)

	mockStore.AssertExpectations(t)
	mockLClient.AssertExpectations(t)

}

func TestCreateGenerated(t *testing.T) {

	// Mocks
	mockStore := new(mocks.Store)
	mockLClient := new(mocks.LightningClient)
	mockLClient.On("GetInfo", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*lnrpc.GetInfoRequest")).Once().Return(&lnrpc.GetInfoResponse{IdentityPubkey: "testing"}, nil)
	mockDCache := new(mocks.DistCache)

	// Bootstrap authentication
	account := &tdrpc.Account{
		Id:      "123123123123132132132123131123123123123132132132123131123123123333",
		Address: "2MsoezssHTCZbeoVcZ5NgYmtNiUpyzAc5hm",
	}
	ctx := addAccount(context.Background(), account)

	mockLClient.On("AddInvoice", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*lnrpc.Invoice")).Once().Return(
		&lnrpc.AddInvoiceResponse{
			RHash:          []byte("asdfasdfasdf"),
			PaymentRequest: "whutwhute",
		}, nil,
	)
	mockStore.On("ProcessLedgerRecord", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*tdrpc.LedgerRecord")).Once().Return(nil)
	mockStore.On("GetActiveGeneratedLightningLedgerRequest", mock.AnythingOfType("*context.valueCtx"), account.Id).Once().Return(nil, store.ErrNotFound)

	// RPC Server
	s, err := NewTDRPCServer(mockStore, mockLClient, mockDCache)
	assert.Nil(t, err)

	// Bad Value
	r, err := s.CreateGenerated(ctx, nil)
	assert.Nil(t, err)
	assert.Equal(t, "whutwhute", r.Request)

	mockStore.AssertExpectations(t)
	mockLClient.AssertExpectations(t)

}

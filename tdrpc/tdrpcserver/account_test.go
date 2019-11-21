package tdrpcserver

import (
	"context"
	"testing"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"git.coinninja.net/backend/thunderdome/mocks"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func TestGetAccount(t *testing.T) {

	// Mocks
	mockStore := new(mocks.Store)
	mockLClient := new(mocks.LightningClient)
	mockLClient.On("GetInfo", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*lnrpc.GetInfoRequest")).Once().Return(&lnrpc.GetInfoResponse{IdentityPubkey: "testing"}, nil)
	mockDCache := new(mocks.DistCache)

	// RPC Server
	s, err := NewTDRPCServer(mockStore, mockLClient, mockDCache)
	assert.Nil(t, err)

	// Create a sample account and put it into the context for the call
	a := &tdrpc.Account{
		Id: "test",
	}

	// Make the request
	b, err := s.GetAccount(addAccount(context.Background(), a), nil)
	assert.Nil(t, err)
	assert.Equal(t, a, b)

	mockStore.AssertExpectations(t)
	mockLClient.AssertExpectations(t)

}

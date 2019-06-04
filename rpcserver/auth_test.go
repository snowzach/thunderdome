package rpcserver

import (
	"context"
	"testing"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"

	"git.coinninja.net/backend/thunderdome/mocks"
	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func TestAuthFuncOverride(t *testing.T) {

	// Mocks
	mockStore := new(mocks.Store)
	mockLClient := new(mocks.LightningClient)
	mockLClient.On("GetInfo", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*lnrpc.GetInfoRequest")).Once().Return(&lnrpc.GetInfoResponse{IdentityPubkey: "testing"}, nil)
	// RPC Server
	s, err := NewRPCServer(mockStore, mockLClient)
	assert.Nil(t, err)

	// Bad Value
	_, err = s.AuthFuncOverride(metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "SOME BAD VALUE")), "test")
	assert.NotNil(t, err)

	pubKey := "123123123123132132132123131123123123123132132132123131123123123333"
	address := "2MsoezssHTCZbeoVcZ5NgYmtNiUpyzAc5hm"

	// Not Found - creeate new account and address
	mockStore.On("AccountGetByID", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("string")).Once().Return(nil, store.ErrNotFound)
	mockLClient.On("NewAddress", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*lnrpc.NewAddressRequest")).Once().Return(&lnrpc.NewAddressResponse{Address: address}, nil)
	mockStore.On("AccountSave", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*tdrpc.Account")).Once().
		Return(func(ctx context.Context, a *tdrpc.Account) *tdrpc.Account { return a }, nil)
	ctx, err := s.AuthFuncOverride(metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", pubKey)), "test")
	assert.Nil(t, err)
	account := getAccount(ctx)
	assert.NotNil(t, account)
	assert.Equal(t, AccountTypePubKey+":"+pubKey, account.Id)
	assert.Equal(t, address, account.Address)

	// Make the request
	mockStore.On("AccountGetByID", mock.AnythingOfType("*context.valueCtx"), account.Id).Once().Return(account, nil)
	ctx, err = s.AuthFuncOverride(metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", pubKey)), "test")
	assert.Nil(t, err)
	account = getAccount(ctx)
	assert.NotNil(t, account)
	assert.Equal(t, AccountTypePubKey+":"+pubKey, account.Id)
	assert.Equal(t, address, account.Address)

	mockStore.AssertExpectations(t)
	mockLClient.AssertExpectations(t)

}

package tdrpcserver

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
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
	mockDCache := new(mocks.DistCache)

	// RPC Server
	s, err := newTDRPCServer(mockStore, mockLClient, mockDCache)
	assert.Nil(t, err)

	// Bad Value
	_, err = s.AuthFuncOverride(metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		tdrpc.MetadataAuthPubKeyString, "SOME BAD PUBKEY",
		tdrpc.MetadataAuthSignature, "Bad Signature",
		tdrpc.MetadataAuthTimestamp, time.Now().Format(time.RFC3339),
	)), "test")
	assert.NotNil(t, err)

	// Signature Stuff
	key, err := NewKey()
	assert.Nil(t, err)
	pubKey := HexEncodedPublicKey(key)
	timeString := time.Now().UTC().Format(time.RFC3339)
	sig, err := key.Sign(chainhash.DoubleHashB([]byte(timeString)))
	assert.Nil(t, err)
	sigHexString := hex.EncodeToString(sig.Serialize())

	// Not Found - creeate new account and address
	mockStore.On("GetAccountByID", mock.AnythingOfType("*context.valueCtx"), AccountTypePubKey+":"+pubKey).Once().Return(nil, store.ErrNotFound)
	// Valid information but CreatedGenerated endpoint should return error
	_, err = s.AuthFuncOverride(metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		tdrpc.MetadataAuthPubKeyString, pubKey,
		tdrpc.MetadataAuthSignature, sigHexString,
		tdrpc.MetadataAuthTimestamp, timeString,
	)), tdrpc.CreateGeneratedEndpoint)
	assert.NotNil(t, err)

	// Valid information to other endpoint should return a new account
	key, err = NewKey()
	assert.Nil(t, err)
	pubKey = HexEncodedPublicKey(key)
	timeString = time.Now().UTC().Format(time.RFC3339)
	sig, err = key.Sign(chainhash.DoubleHashB([]byte(timeString)))
	assert.Nil(t, err)
	sigHexString = hex.EncodeToString(sig.Serialize())
	address := "2MsoezssHTCZbeoVcZ5NgYmtNiUpyzAc5hm"
	mockStore.On("GetAccountByID", mock.AnythingOfType("*context.valueCtx"), AccountTypePubKey+":"+pubKey).Once().Return(nil, store.ErrNotFound)
	mockLClient.On("NewAddress", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*lnrpc.NewAddressRequest")).Once().Return(&lnrpc.NewAddressResponse{Address: address}, nil)
	mockStore.On("SaveAccount", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*tdrpc.Account")).Once().
		Return(func(ctx context.Context, a *tdrpc.Account) *tdrpc.Account { return a }, nil)
	ctx, err := s.AuthFuncOverride(metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		tdrpc.MetadataAuthPubKeyString, pubKey,
		tdrpc.MetadataAuthSignature, sigHexString,
		tdrpc.MetadataAuthTimestamp, timeString,
	)), "test")
	assert.Nil(t, err)

	account := getAccount(ctx)
	assert.NotNil(t, account)
	assert.Equal(t, AccountTypePubKey+":"+pubKey, account.Id)
	assert.Equal(t, address, account.Address)

	// Make the request
	mockStore.On("GetAccountByID", mock.AnythingOfType("*context.valueCtx"), account.Id).Once().Return(account, nil)
	ctx, err = s.AuthFuncOverride(metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		tdrpc.MetadataAuthPubKeyString, pubKey,
		tdrpc.MetadataAuthSignature, sigHexString,
		tdrpc.MetadataAuthTimestamp, timeString,
	)), "test")
	assert.Nil(t, err)
	account = getAccount(ctx)
	assert.NotNil(t, account)
	assert.Equal(t, AccountTypePubKey+":"+pubKey, account.Id)
	assert.Equal(t, address, account.Address)

	mockStore.AssertExpectations(t)
	mockLClient.AssertExpectations(t)

}

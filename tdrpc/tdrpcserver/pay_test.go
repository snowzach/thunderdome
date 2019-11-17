package tdrpcserver

import (
	"context"
	"testing"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	_ "git.coinninja.net/backend/thunderdome/conf"
	"git.coinninja.net/backend/thunderdome/mocks"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func TestPay(t *testing.T) {

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

	// Decoded payment request
	pr := &lnrpc.PayReq{
		Destination: "test",
		Expiry:      time.Now().Add(time.Hour).Unix(),
	}
	mockLClient.On("DecodePayReq", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*lnrpc.PayReqString")).Once().Return(pr, nil)

	// Route/Fee requests
	route := &lnrpc.Route{
		TotalFees: 123,
	}
	mockLClient.On("QueryRoutes", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*lnrpc.QueryRoutesRequest")).Once().Return(&lnrpc.QueryRoutesResponse{Routes: []*lnrpc.Route{route}}, nil)

	mockStore.On("ProcessLedgerRecord", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*tdrpc.LedgerRecord")).Once().Return(nil)
	mockLClient.On("SendPaymentSync", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*lnrpc.SendRequest")).Once().
		Return(&lnrpc.SendResponse{}, nil)
	mockStore.On("ProcessLedgerRecord", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*tdrpc.LedgerRecord")).Once().Return(nil)

	// Insufficient funds
	_, err = s.Pay(ctx, &tdrpc.PayRequest{
		Request: "somerequest",
		Value:   20,
	})
	assert.Nil(t, err)

	mockStore.AssertExpectations(t)
	mockLClient.AssertExpectations(t)

}

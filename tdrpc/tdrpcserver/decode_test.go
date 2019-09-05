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

func TestDecode(t *testing.T) {

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
	}
	ctx := addAccount(context.Background(), account)

	// This is what lnd returns
	pr := &lnrpc.PayReq{
		Destination: "test",
	}

	// This is what we expect
	tdrpcPayReq := &tdrpc.DecodeResponse{
		Destination:     pr.Destination,
		PaymentHash:     pr.PaymentHash,
		NumSatoshis:     pr.NumSatoshis,
		Timestamp:       pr.Timestamp,
		Expiry:          pr.Expiry,
		Description:     pr.Description,
		DescriptionHash: pr.DescriptionHash,
		FallbackAddr:    pr.FallbackAddr,
		CltvExpiry:      pr.CltvExpiry,
		RouteHints:      []*tdrpc.RouteHint{}, // TODO: Decode route hints
	}

	// AddInvoice call
	mockLClient.On("DecodePayReq", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*lnrpc.PayReqString")).Once().Return(pr, nil)

	// Make the request
	response, err := s.Decode(ctx, &tdrpc.DecodeRequest{
		Request: "lnbcrt100n1pw0ry32pp537g0nunvpgv0xvuqdejl0j6nsykt7s4mrxkflfv97272xln6xtcsdqjfpjkcmr0yptk7unvvscqzpgxqyz5vql9vf88y47hnx9pfk6nu54e0zhh9rfmluqk8xq7jckyahltcm24gjps4mjje7ceznxsve5jum9lkrq28sjyqgxh8pp3xq7atf6d3pkhsp53kfed",
	})
	assert.Nil(t, err)
	assert.Equal(t, tdrpcPayReq, response)

	mockStore.AssertExpectations(t)
	mockLClient.AssertExpectations(t)

}

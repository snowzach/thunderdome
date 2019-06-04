package rpcserver

import (
	"context"
	"fmt"

	"github.com/lightningnetwork/lnd/lnrpc"
	"go.uber.org/zap"

	"git.coinninja.net/backend/thunderdome/tdrpc"
	"git.coinninja.net/backend/thunderdome/thunderdome"
)

type RPCServer struct {
	logger   *zap.SugaredLogger
	store    thunderdome.Store
	myPubKey string
	lclient  lnrpc.LightningClient
}

type contextKey string

const (
	contextKeyAccount = "account"
)

// addAccount will include the authenticated account to the RPC context
func addAccount(ctx context.Context, account *tdrpc.Account) context.Context {
	return context.WithValue(ctx, contextKey(contextKeyAccount), account)
}

// getAccount is a helper to get the Authenticated account from the RPC context (returning nil if not found)
func getAccount(ctx context.Context) *tdrpc.Account {
	account, ok := ctx.Value(contextKey(contextKeyAccount)).(*tdrpc.Account)
	if ok {
		return account
	}
	return nil
}

// NewRPCServer creates the server
func NewRPCServer(store thunderdome.Store, lclient lnrpc.LightningClient) (*RPCServer, error) {

	info, err := lclient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("Could not test lightning connection: %v", err)
	}

	// Return the server
	return &RPCServer{
		logger:   zap.S().With("package", "rpcserver"),
		store:    store,
		myPubKey: info.IdentityPubkey,
		lclient:  lclient,
	}, nil

}

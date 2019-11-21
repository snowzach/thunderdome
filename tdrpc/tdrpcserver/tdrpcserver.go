package tdrpcserver

import (
	"context"
	"fmt"

	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
	"go.uber.org/zap"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

type tdRPCServer struct {
	logger   *zap.SugaredLogger
	store    tdrpc.Store
	cache    store.DistCache
	myPubKey string
	lclient  lnrpc.LightningClient
}

type contextKey string

const (
	contextKeyAccount = "account"
	contextKeyAgent   = "agent"
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

// Returns if it's being called on behalf of the agent
func isAgent(ctx context.Context) bool {
	agent, ok := ctx.Value(contextKey(contextKeyAgent)).(bool)
	if ok {
		return agent
	}
	return false
}

// NewTDRPCServer creates the server
func NewTDRPCServer(store tdrpc.Store, lclient lnrpc.LightningClient, cache store.DistCache) (tdrpc.ThunderdomeRPCServer, error) {

	return newTDRPCServer(store, lclient, cache)

}

func newTDRPCServer(store tdrpc.Store, lclient lnrpc.LightningClient, cache store.DistCache) (*tdRPCServer, error) {

	info, err := lclient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("Could not test lightning connection: %v", err)
	}

	// Return the server
	s := &tdRPCServer{
		logger:   zap.S().With("package", "tdrpc"),
		store:    store,
		cache:    cache,
		myPubKey: info.IdentityPubkey,
		lclient:  lclient,
	}

	if config.GetBool("tdome.disable_auth") {
		s.logger.Warn("*** WARNING *** AUTH IS DISABLED *** WARNING ***")
	}

	return s, nil

}

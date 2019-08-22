package tdrpcserver

import (
	"context"
	"fmt"

	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
	"go.uber.org/zap"

	"git.coinninja.net/backend/thunderdome/tdrpc"
	"git.coinninja.net/backend/thunderdome/thunderdome"
)

type tdRPCServer struct {
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

// NewTDRPCServer creates the server
func NewTDRPCServer(store thunderdome.Store, lclient lnrpc.LightningClient) (tdrpc.ThunderdomeRPCServer, error) {

	return newTDRPCServer(store, lclient)

}

func newTDRPCServer(store thunderdome.Store, lclient lnrpc.LightningClient) (*tdRPCServer, error) {

	info, err := lclient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("Could not test lightning connection: %v", err)
	}

	// Return the server
	s := &tdRPCServer{
		logger:   zap.S().With("package", "rpcserver"),
		store:    store,
		myPubKey: info.IdentityPubkey,
		lclient:  lclient,
	}

	if config.GetBool("tdome.disable_auth") {
		s.logger.Warn("*** WARNING *** AUTH IS DISABLED *** WARNING ***")
	}

	return s, nil

}

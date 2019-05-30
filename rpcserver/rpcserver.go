package rpcserver

import (
	"context"

	"github.com/lightningnetwork/lnd/lnrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"git.coinninja.net/backend/thunderdome/tdrpc"
	"git.coinninja.net/backend/thunderdome/thunderdome"
)

type RPCServer struct {
	logger   *zap.SugaredLogger
	store    thunderdome.Store
	myPubKey string
	conn     *grpc.ClientConn
	lclient  lnrpc.LightningClient
}

type contextKey string

const contextKeyAccount = "account"

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
func NewRPCServer(store thunderdome.Store, conn *grpc.ClientConn) (*RPCServer, error) {

	// Fetch the node info to make sure we know our own identity for self-payments
	lclient := lnrpc.NewLightningClient(conn)
	info, err := lclient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	// Return the server
	return &RPCServer{
		logger:   zap.S().With("package", "rpcserver"),
		store:    store,
		myPubKey: info.IdentityPubkey,
		conn:     conn,
		lclient:  lclient,
	}, nil

}

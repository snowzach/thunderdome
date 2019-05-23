package rpcserver

import (
	"context"

	"github.com/lightningnetwork/lnd/lnrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

type RPCStore interface {
	AccountGetByID(context.Context, string) (*tdrpc.Account, error)
	AccountSave(context.Context, *tdrpc.Account) (*tdrpc.Account, error)
	UpsertLedgerRecord(context.Context, *tdrpc.LedgerRecord) error
}

type RPCServer struct {
	logger   *zap.SugaredLogger
	rpcStore RPCStore

	conn    *grpc.ClientConn
	lclient lnrpc.LightningClient
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
func NewRPCServer(rpcStore RPCStore, conn *grpc.ClientConn) (*RPCServer, error) {

	return &RPCServer{
		logger:   zap.S().With("package", "rpcserver"),
		rpcStore: rpcStore,

		conn:    conn,
		lclient: lnrpc.NewLightningClient(conn),
	}, nil

}

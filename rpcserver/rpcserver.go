package rpcserver

import (
	"context"

	"github.com/lightningnetwork/lnd/lnrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

type RPCStore interface {
	AccountGetByID(ctx context.Context, accountID string) (*tdrpc.Account, error)
	AccountSave(ctx context.Context, account *tdrpc.Account) (*tdrpc.Account, error)
	ProcessLedgerRecord(ctx context.Context, lr *tdrpc.LedgerRecord) error
	ProcessInternal(ctx context.Context, id string) (*tdrpc.LedgerRecord, error)
	GetLedger(ctx context.Context, accountID string) ([]*tdrpc.LedgerRecord, error)
	GetLedgerRecord(ctx context.Context, id string, direction tdrpc.LedgerRecord_Direction) (*tdrpc.LedgerRecord, error)
}

type RPCServer struct {
	logger   *zap.SugaredLogger
	rpcStore RPCStore
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
func NewRPCServer(rpcStore RPCStore, conn *grpc.ClientConn) (*RPCServer, error) {

	// Fetch the node info to make sure we know our own identity for self-payments
	lclient := lnrpc.NewLightningClient(conn)
	info, err := lclient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	// Return the server
	return &RPCServer{
		logger:   zap.S().With("package", "rpcserver"),
		rpcStore: rpcStore,
		myPubKey: info.IdentityPubkey,
		conn:     conn,
		lclient:  lclient,
	}, nil

}

package rpcserver

import (
	"context"
	"regexp"

	"github.com/lightningnetwork/lnd/lnrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

var (
	uuidRegexp   = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	pubkeyRegexp = regexp.MustCompile("^[a-f0-9]{66}$")
)

type RPCStore interface {
	UserGetByID(ctx context.Context, id string) (*tdrpc.User, error)
	UserGetByLogin(ctx context.Context, login string) (*tdrpc.User, error)
	UserSave(ctx context.Context, user *tdrpc.User) (*tdrpc.User, error)
	AddInvoice(ctx context.Context, userID string, paymentHash string) error
	GetUserIDByPaymentHash(ctx context.Context, paymentHash string) (string, error)
}

type RPCServer struct {
	logger   *zap.SugaredLogger
	rpcStore RPCStore

	conn    *grpc.ClientConn
	lclient lnrpc.LightningClient
}

type contextKey string

// NewServer creates the server
func NewRPCServer(rpcStore RPCStore, conn *grpc.ClientConn) (*RPCServer, error) {

	return &RPCServer{
		logger:   zap.S().With("package", "rpcserver"),
		rpcStore: rpcStore,

		conn:    conn,
		lclient: lnrpc.NewLightningClient(conn),
	}, nil

}

// AuthFuncOverride will handle authentication
func (s *RPCServer) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {

	// // Bypass for Login
	if fullMethodName == "/tdrpc.ThunderdomeRPC/Login" {
		return ctx, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, grpc.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	a := md.Get("authorization")
	if len(a) != 1 {
		return ctx, grpc.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	user, err := s.rpcStore.UserGetByID(ctx, a[0])
	if err == store.ErrNotFound {
		return ctx, grpc.Errorf(codes.PermissionDenied, "Permission Denied")
	} else if err != nil {
		return ctx, grpc.Errorf(codes.Internal, "UserGetByID Error: %v", err)
	}

	// Include the user in the context
	return addUser(ctx, user), nil
}

// addUser will include the authenticated user to the RPC context
func addUser(ctx context.Context, user *tdrpc.User) context.Context {
	return context.WithValue(ctx, contextKey("user"), user)
}

// getUser is a helper to get the Authenticated user from the RPC context (returning nil if not found)
func getUser(ctx context.Context) *tdrpc.User {
	user, ok := ctx.Value(contextKey("user")).(*tdrpc.User)
	if ok {
		return user
	}
	return nil
}

package adminrpcserver

import (
	"context"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	config "github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/cnauth"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

type contextKey string

const (
	contextKeyRole = "role"
)

// AuthFuncOverride will handle authentication
func (s *adminRPCServer) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {

	if s.cnAuthClient == nil {
		s.logger.Warn("*** ADMIN REQUEST - AUTH DISABLED ***")
		return ctx, nil
	}

	jwt, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return ctx, status.Errorf(codes.PermissionDenied, "missing auth method")
	}

	// Verify the user role
	_, role, err := s.cnAuthClient.VerifyGetRole(ctx, jwt, config.GetString("tdome.firebase_admin_role"))
	if err != nil {
		s.logger.Errorw("VerifyGetRole Failed", "error", err)
		return ctx, tdrpc.ErrInvalidLogin
	}

	// Make sure the user has at least Read access
	hasRole, err := cnauth.HasRole(role, cnauth.RoleRead)
	if err != nil {
		return ctx, status.Errorf(codes.PermissionDenied, "role error: %v", err)
	}
	if !hasRole {
		return ctx, tdrpc.ErrPermissionDenied
	}

	return addRole(ctx, role), nil

}

// addRole will include the authenticated role to the RPC context
func addRole(ctx context.Context, role cnauth.Role) context.Context {
	return context.WithValue(ctx, contextKey(contextKeyRole), role)
}

// getAccount is a helper to get the Authenticated account from the RPC context (returning nil if not found)
func getRole(ctx context.Context) cnauth.Role {
	role, ok := ctx.Value(contextKey(contextKeyRole)).(cnauth.Role)
	if ok {
		return role
	}
	return cnauth.RoleUnknown
}

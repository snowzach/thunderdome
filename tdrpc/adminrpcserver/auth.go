package adminrpcserver

import (
	"context"

	"git.coinninja.net/backend/cnauth"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	_, err = s.cnAuthClient.VerifyRole(ctx, jwt, cnauth.ClaimRolePrefix, cnauth.RoleWrite)
	if err != nil {
		return ctx, status.Errorf(codes.PermissionDenied, "access denied: %v", err)
	}

	return ctx, nil

}

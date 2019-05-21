package rpcserver

import (
	"context"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// Login will take the user's login (public key) and return a User
func (s *RPCServer) Login(ctx context.Context, request *tdrpc.LoginRequest) (*tdrpc.User, error) {

	// Right now the login must be a public key
	if !pubkeyRegexp.MatchString(request.Login) {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid Login")
	}

	// See if we have the user
	user, err := s.rpcStore.UserGetByLogin(ctx, request.Login)
	if err == nil {
		return user, nil
	} else if err != store.ErrNotFound {
		return nil, grpc.Errorf(codes.Internal, "UserGetByLogin Error: %v", err)
	}

	// Otherwise the user doesn't exist, create a new one
	user = new(tdrpc.User)
	user.Login = request.Login

	// Fetch an unused address from the lightning node
	address, err := s.lclient.NewAddress(ctx, &lnrpc.NewAddressRequest{
		Type: lnrpc.AddressType_WITNESS_PUBKEY_HASH,
	})
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "New Address Error: %v", err)
	}

	// Save the user
	user.Address = address.Address
	user, err = s.rpcStore.UserSave(ctx, user)
	if err != nil {
		return nil, grpc.Errorf(codes.Internal, "UserSave Error: %v", err)
	}

	return user, nil

}

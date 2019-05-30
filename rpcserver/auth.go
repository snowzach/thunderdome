package rpcserver

import (
	"context"
	"regexp"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

const (
	AccountTypePubKey = "pubkey"
)

var (
	pubkeyRegexp = regexp.MustCompile("^[a-f0-9]{66}$")
)

// AuthFuncOverride will handle authentication
func (s *RPCServer) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, grpc.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	// The authorization header is the publickey
	a := md.Get("authorization")
	if len(a) != 1 {
		return ctx, grpc.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	// The accountID will account for different methods of logging in, right now we support public key
	var accountID string
	if pubkeyRegexp.MatchString(a[0]) {
		// The account ID is prefix:value
		accountID = AccountTypePubKey + ":" + a[0]
		// PERFORM AUTH
	} else {
		return nil, grpc.Errorf(codes.InvalidArgument, "Invalid Login")
	}

	// See if we have an account already?
	account, err := s.store.AccountGetByID(ctx, accountID)
	if err == store.ErrNotFound {

		// Create a new account
		account = new(tdrpc.Account)
		account.Id = accountID

		// Fetch an unused address from the lightning node
		address, err := s.lclient.NewAddress(ctx, &lnrpc.NewAddressRequest{
			Type: lnrpc.AddressType_NESTED_PUBKEY_HASH,
		})
		if err != nil {
			return nil, grpc.Errorf(codes.Internal, "New Address Error: %v", err)
		}

		// Save the account
		account.Address = address.Address
		account, err = s.store.AccountSave(ctx, account)
		if err != nil {
			return nil, grpc.Errorf(codes.Internal, "AccountSave Error: %v", err)
		}

	} else if err != nil {
		return ctx, grpc.Errorf(codes.Internal, "AccountGetByID Error: %v", err)
	}

	// Include the account in the context
	return addAccount(ctx, account), nil

}

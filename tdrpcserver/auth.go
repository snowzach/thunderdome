package tdrpcserver

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
	"git.coinninja.net/backend/thunderdome/thunderdome"
)

const (
	AccountTypePubKey = "pubkey"

	Meta
)

var (
	pubkeyRegexp = regexp.MustCompile("^[a-f0-9]{66}$")
)

// AuthFuncOverride will handle authentication
func (s *tdRPCServer) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {

	// Get request metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, status.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	// Get the user pubKeyString
	pubKeyString := mdfirst(md, thunderdome.MetadataAuthPubKeyString)
	if pubKeyString == "" {
		return ctx, status.Errorf(codes.PermissionDenied, "Invalid Login")
	}

	// Get the timestamp and signature
	ts := mdfirst(md, thunderdome.MetadataAuthTimestamp)
	sig := mdfirst(md, thunderdome.MetadataAuthSignature)

	// If we're calling the CreateGeneric endpoint and the sig is set to the create_generic_secret we can skip auth
	// If the secret is not set, the CreateGeneric endpoint cannot be called
	// This allows trusted sources, like btc-api to generate invoices
	if fullMethodName != tdrpc.CreateGeneratedEndpoint || sig != config.GetString("tdome.create_generic_secret") || config.GetString("tdome.create_generic_secret") == "" {
		// If auth is disabled
		if !config.GetBool("tdome.disable_auth") {
			if ts == "" || sig == "" {
				return ctx, status.Errorf(codes.PermissionDenied, "Permission Denied")
			}
			// Verify the signature
			err := ValidateTimestampSigntature(ts, pubKeyString, sig, time.Now())
			if err != nil {
				return ctx, status.Errorf(codes.PermissionDenied, err.Error())
			}
		}
	}

	// The accountID will account for different methods of logging in, right now we support public key
	var accountID string
	if pubkeyRegexp.MatchString(pubKeyString) {
		// The account ID is prefix:value
		accountID = AccountTypePubKey + ":" + pubKeyString
		// PERFORM AUTH
	} else {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid Login")
	}

	// See if we have an account already?
	account, err := s.store.AccountGetByID(ctx, accountID)
	if err == store.ErrNotFound {

		// If the endpoint is the CreateGeneratedEndpoint we do not want to auto-create an account. Just return a not found error
		if fullMethodName == tdrpc.CreateGeneratedEndpoint {
			return ctx, status.Errorf(codes.NotFound, "account does not exist")
		}

		// Create a new account
		account = new(tdrpc.Account)
		account.Id = accountID

		// Fetch an unused address from the lightning node
		address, err := s.lclient.NewAddress(ctx, &lnrpc.NewAddressRequest{
			Type: lnrpc.AddressType_NESTED_PUBKEY_HASH,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "New Address Error: %v", err)
		}

		// Save the account
		account.Address = address.Address
		account, err = s.store.AccountSave(ctx, account)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "AccountSave Error: %v", err)
		}

	} else if err != nil {
		return ctx, status.Errorf(codes.Internal, "AccountGetByID Error: %v", err)
	}

	// Include the account in the context
	return addAccount(ctx, account), nil

}

func mdfirst(md metadata.MD, key string) string {
	val := md.Get(strings.ToLower(key))
	if len(val) > 0 {
		return val[0]
	}
	return ""
}
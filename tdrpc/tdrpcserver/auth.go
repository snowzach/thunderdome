package tdrpcserver

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

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
func (s *tdRPCServer) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {

	// Returns a 503
	if config.GetBool("tdome.disabled") {
		return ctx, tdrpc.ErrServiceUnavailable
	}

	// No auth required for DecodeEndpoint
	if fullMethodName == tdrpc.DecodeEndpoint {
		return ctx, nil
	}

	// Get request metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, tdrpc.ErrInvalidLogin
	}

	// Get the user pubKeyString
	pubKeyString := mdfirst(md, tdrpc.MetadataAuthPubKeyString)
	if pubKeyString == "" {
		return ctx, tdrpc.ErrInvalidLogin
	}

	// Get the timestamp and signature
	ts := mdfirst(md, tdrpc.MetadataAuthTimestamp)
	sig := mdfirst(md, tdrpc.MetadataAuthSignature)
	nonce := mdfirst(md, tdrpc.MetadataAuthNonce)

	// This handles authentication, there are several cases, break from the for loop when Authenticated
	for {
		// Auth is disabled
		if config.GetBool("tdome.disable_auth") {
			break // Authenticated
		}

		// If this is an agent and it's going to an allowed endpoint
		if allowAgent(fullMethodName, sig) {
			// Add the agent flag to the context
			ctx = context.WithValue(ctx, contextKey(contextKeyAgent), true)
			break // Authenticated
		}

		// Otherwise require a valid signature
		err := ValidateTimestampAndNonceSigntature(ts, nonce, pubKeyString, sig, time.Now())
		if err != nil {
			return ctx, err
		}

		break //nolint - We are authenticated
	}

	// The accountID will account for different methods of logging in, right now we support public key
	var accountID string
	if pubkeyRegexp.MatchString(pubKeyString) {
		// The account ID is prefix:value
		accountID = AccountTypePubKey + ":" + pubKeyString
		// PERFORM AUTH
	} else {
		return nil, tdrpc.ErrInvalidLogin
	}

	// Check the nonce and see if it's been used already
	if nonce != "" {
		exists, err := s.cache.Exists("nonce", accountID+":"+nonce)
		if err != nil {
			s.logger.Errorw("DistCache Exists Error", "error", err, "nonce", accountID+":"+nonce)
			return nil, status.Errorf(codes.Internal, "DistCache Error: %v", err)
		}

		// This Nonce has been used already
		if exists {
			return ctx, tdrpc.ErrInvalidLogin
		}

		// Create the nonce
		err = s.cache.Set("nonce", accountID+":"+nonce, 1, 20*time.Minute)
		if err != nil {
			s.logger.Errorw("DistCache Set Error", "error", err, "nonce", accountID+":"+nonce)
		}
	}

	// See if we have an account already?
	account, err := s.store.GetAccountByID(ctx, accountID)
	if err == store.ErrNotFound {

		// We will never auto-create an account for the CreateGeneratedEndpoint
		if fullMethodName == tdrpc.CreateGeneratedEndpoint {
			return ctx, tdrpc.ErrNotFound
		}

		// Create a new account
		account = new(tdrpc.Account)
		account.Id = accountID
		account.Locked = config.GetBool("tdome.lock_new_accounts")

		// Fetch an unused address from the lightning node
		address, err := s.lclient.NewAddress(ctx, &lnrpc.NewAddressRequest{
			Type: lnrpc.AddressType_NESTED_PUBKEY_HASH,
		})
		if err != nil {
			s.logger.Errorw("LND NewAddress Error", "error", err)
			return nil, status.Errorf(codes.Internal, "New Address Error: %v", err)
		}

		// Save the account
		account.Address = address.Address
		account, err = s.store.SaveAccount(ctx, account)
		if err != nil {
			s.logger.Errorw("SaveAccount Error", zap.Any("account", account), "error", err)
			return nil, status.Errorf(codes.Internal, "SaveAccount internal error")
		}

	} else if err != nil {
		s.logger.Errorw("GetAccountByID Error", "account_id", accountID, "error", err)
		return nil, status.Errorf(codes.Internal, "GetAccountByID internal error")
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

func allowAgent(endpoint string, sig string) bool {

	// We allow the following endpoints from the agent
	switch endpoint {
	case tdrpc.CreateGeneratedEndpoint:
	case tdrpc.PayEndpoint:
	case tdrpc.GetPreAuthEndpoint:
	default:
		return false
	}

	// The signature must match the agent_secret
	if sig != "" && sig == config.GetString("tdome.agent_secret") {
		return true
	}

	return false
}

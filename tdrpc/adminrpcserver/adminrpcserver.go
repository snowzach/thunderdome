package adminrpcserver

import (
	"git.coinninja.net/backend/cnauth"
	"go.uber.org/zap"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

type adminRPCServer struct {
	logger       *zap.SugaredLogger
	store        tdrpc.Store
	cnAuthClient *cnauth.Client
}

// NewAdminRPCServer creates the server
func NewAdminRPCServer(store tdrpc.Store, cnAuthClient *cnauth.Client) (tdrpc.AdminRPCServer, error) {

	return newAdminRPCServer(store, cnAuthClient)

}

func newAdminRPCServer(store tdrpc.Store, cnAuthClient *cnauth.Client) (*adminRPCServer, error) {

	// Return the server
	s := &adminRPCServer{
		logger:       zap.S().With("package", "adminrpc"),
		store:        store,
		cnAuthClient: cnAuthClient,
	}

	if cnAuthClient == nil {
		s.logger.Warn("*** WARNING *** AUTH IS DISABLED *** WARNING ***")
	}

	return s, nil

}

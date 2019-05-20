package thunderdome

import (
	"context"
	"regexp"

	"github.com/lightningnetwork/lnd/lnrpc"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

var (
	uuidRegexp   = regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	pubkeyRegexp = regexp.MustCompile("^[a-f0-9]{66}$")
)

type Store interface {
	UserGetByID(ctx context.Context, id string) (*tdrpc.User, error)
	UserGetByLogin(ctx context.Context, login string) (*tdrpc.User, error)
	UserSave(ctx context.Context, user *tdrpc.User) (*tdrpc.User, error)
}

type Server struct {
	store   Store
	lclient lnrpc.LightningClient
}

func NewServer(store Store, lclient lnrpc.LightningClient) (*Server, error) {

	return &Server{
		store:   store,
		lclient: lclient,
	}, nil

}

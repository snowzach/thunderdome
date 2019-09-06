package versionrpcserver

import (
	"context"

	emptypb "github.com/golang/protobuf/ptypes/empty"

	"git.coinninja.net/backend/thunderdome/conf"
	"git.coinninja.net/backend/thunderdome/server/versionrpc"
)

type versionRPCServer struct{}

// New returns a new version server
func New() versionrpc.VersionRPCServer {
	return versionRPCServer{}
}

// Version returns the version
func (vs versionRPCServer) Version(ctx context.Context, _ *emptypb.Empty) (*versionrpc.VersionResponse, error) {

	return &versionrpc.VersionResponse{
		Version: conf.GitVersion,
	}, nil

}

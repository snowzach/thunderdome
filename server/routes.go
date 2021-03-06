package server

import (
	"context"
	"net/http"

	assetfs "github.com/elazarl/go-bindata-assetfs"

	"git.coinninja.net/backend/thunderdome/embed"
	"git.coinninja.net/backend/thunderdome/server/versionrpc"
	"git.coinninja.net/backend/thunderdome/server/versionrpc/versionrpcserver"
)

// SetupRoutes configures all the routes for this service
func (s *Server) SetupRoutes() {

	// Register our routes - you need at aleast one route
	s.router.Get("/none", func(w http.ResponseWriter, r *http.Request) {})

	// Register RPC Services
	versionrpc.RegisterVersionRPCServer(s.GRPCServer(), versionrpcserver.New())
	s.GWReg(versionrpc.RegisterVersionRPCHandlerFromEndpoint)

	// Serve api-docs and swagger-ui
	fs := http.FileServer(&assetfs.AssetFS{Asset: embed.Asset, AssetDir: embed.AssetDir, AssetInfo: embed.AssetInfo, Prefix: "public"})
	s.router.Get("/api-docs/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
	s.router.Get("/swagger-ui/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))
	s.router.Get("/test/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}))

}

// AuthFuncOverride any server related functions will not require auth
func (s *Server) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, nil
}

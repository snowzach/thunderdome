package tdrpcserver

import (
	"context"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// DecodePayReq passes through decoding the pay request
func (s *tdRPCServer) Decode(ctx context.Context, request *tdrpc.DecodeRequest) (*tdrpc.DecodeResponse, error) {

	// Decode and return the PayRequest
	pr, err := s.lclient.DecodePayReq(ctx, &lnrpc.PayReqString{PayReq: request.Request})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Could not DecodePayReq: %v", status.Convert(err).Message())
	}

	// Convert it to our type
	return &tdrpc.DecodeResponse{
		Destination:     pr.Destination,
		PaymentHash:     pr.PaymentHash,
		NumSatoshis:     pr.NumSatoshis,
		Timestamp:       pr.Timestamp,
		Expiry:          pr.Expiry,
		Description:     pr.Description,
		DescriptionHash: pr.DescriptionHash,
		FallbackAddr:    pr.FallbackAddr,
		CltvExpiry:      pr.CltvExpiry,
		RouteHints:      []*tdrpc.RouteHint{}, // TODO: Decode route hints
	}, err

}

package server

import (
	"context"
	"fmt"

	pb "github.com/runmachine-io/runmachine/pkg/resource/proto"
)

var (
	ErrUuidRequired = fmt.Errorf("uuid is required")
)

// ProviderGet looks up a provider by UUID and returns a Provider
// protobuf message.
func (s *Server) ProviderGet(
	ctx context.Context,
	req *pb.ProviderGetRequest,
) (*pb.Provider, error) {
	if req.Uuid == "" {
		return nil, ErrUuidRequired
	}
	return &pb.Provider{
		Uuid: "fake",
	}, nil
}

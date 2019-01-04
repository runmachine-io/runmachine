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

// ProviderCreate creates a new provider record in backend storage
func (s *Server) ProviderCreate(
	ctx context.Context,
	req *pb.ProviderCreateRequest,
) (*pb.ProviderCreateResponse, error) {
	rec, err := s.store.ProviderCreate(req.Provider)
	if err != nil {
		return nil, err
	}
	return &pb.ProviderCreateResponse{
		Provider: rec.Provider,
	}, nil
}

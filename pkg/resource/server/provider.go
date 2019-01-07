package server

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/resource/proto"
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
	rec, err := s.store.ProviderGetByUuid(req.Uuid)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		s.log.ERR(
			"failed to get provider with UUID %s from storage: %s",
			req.Uuid, err,
		)
		return nil, ErrUnknown
	}
	return rec.Provider, nil
}

// ProviderCreate creates a new provider record in backend storage
func (s *Server) ProviderCreate(
	ctx context.Context,
	req *pb.ProviderCreateRequest,
) (*pb.ProviderCreateResponse, error) {
	rec, err := s.store.ProviderCreate(req.Provider)
	if err != nil {
		if err == errors.ErrDuplicate {
			return nil, ErrDuplicate
		}
		return nil, err
	}
	return &pb.ProviderCreateResponse{
		Provider: rec.Provider,
	}, nil
}

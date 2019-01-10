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

// ProviderList streams zero or more Provider objects back to the client that
// match a set of optional filters
func (s *Server) ProviderList(
	req *pb.ProviderListRequest,
	stream pb.RunmResource_ProviderListServer,
) error {
	objs, err := s.store.ProvidersGetMatching(req.Any)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		if err = stream.Send(obj.Provider); err != nil {
			return err
		}
	}
	return nil
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

package server

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

func (s *Server) ProviderTypeGet(
	ctx context.Context,
	req *pb.ProviderTypeGetRequest,
) (*pb.ProviderType, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}

	if req.Filter == nil || req.Filter.CodeFilter == nil {
		return nil, ErrCodeFilterRequired
	}
	obj, err := s.store.ProviderTypeGet(req.Filter.CodeFilter.Code)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve provider type of %s: %s",
			req.Filter.CodeFilter,
			err,
		)
		return nil, ErrUnknown
	}
	return obj, nil
}

func (s *Server) ProviderTypeList(
	req *pb.ProviderTypeListRequest,
	stream pb.RunmMetadata_ProviderTypeListServer,
) error {
	if err := checkSession(req.Session); err != nil {
		return err
	}

	objs, err := s.store.ProviderTypeList(req.Any)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		if err = stream.Send(obj); err != nil {
			return err
		}
	}
	return nil
}

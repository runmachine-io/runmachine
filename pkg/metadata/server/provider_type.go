package server

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/proto"
)

// ProviderTypeGetByCode returns an ProviderType protobuffer message with the
// given code. If no such provider type could be found, returns ErrNotFound
func (s *Server) ProviderTypeGetByCode(
	ctx context.Context,
	req *pb.ProviderTypeGetByCodeRequest,
) (*pb.ProviderType, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}
	code := req.Code
	if code == "" {
		return nil, ErrCodeRequired
	}
	obj, err := s.store.ProviderTypeGetByCode(code)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve provider type of %s: %s",
			code, err,
		)
		return nil, ErrUnknown
	}
	return obj, nil
}

// ProviderTypeList streams zero or more ProviderType protobuffer messages back
// to the client that match any of the filters specified in the request payload
func (s *Server) ProviderTypeList(
	req *pb.ProviderTypeListRequest,
	stream pb.RunmMetadata_ProviderTypeListServer,
) error {
	if err := s.checkSession(req.Session); err != nil {
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

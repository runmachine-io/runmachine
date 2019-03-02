package server

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/proto"
)

// ObjectTypeGetByCode returns an ObjectType protobuffer message with the given
// code. If no such object type could be found, returns ErrNotFound
func (s *Server) ObjectTypeGetByCode(
	ctx context.Context,
	req *pb.ObjectTypeGetByCodeRequest,
) (*pb.ObjectType, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}
	code := req.Code
	if code == "" {
		return nil, ErrCodeRequired
	}
	obj, err := s.store.ObjectTypeGetByCode(code)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve object type of %s: %s",
			code, err,
		)
		return nil, ErrUnknown
	}
	return obj, nil
}

// ObjectTypeList streams zero or more ObjectType protobuffer messages back to
// the client that match any of the filters specified in the request payload
func (s *Server) ObjectTypeList(
	req *pb.ObjectTypeListRequest,
	stream pb.RunmMetadata_ObjectTypeListServer,
) error {
	if err := s.checkSession(req.Session); err != nil {
		return err
	}
	objs, err := s.store.ObjectTypeList(req.Any)
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

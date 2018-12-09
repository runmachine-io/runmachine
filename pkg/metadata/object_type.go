package metadata

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/proto"
)

func (s *Server) ObjectTypeGet(
	ctx context.Context,
	req *pb.ObjectTypeGetRequest,
) (*pb.ObjectType, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}

	if req.Filter == nil || req.Filter.Search == "" {
		return nil, ErrCodeRequired
	}
	obj, err := s.store.ObjectTypeGet(req.Filter.Search)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve object type of %s: %s",
			req.Filter.Search,
			err,
		)
		return nil, ErrUnknown
	}
	return obj, nil
}

func (s *Server) ObjectTypeList(
	req *pb.ObjectTypeListRequest,
	stream pb.RunmMetadata_ObjectTypeListServer,
) error {
	if err := checkSession(req.Session); err != nil {
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

package metadata

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/proto"
)

var (
	ErrCodeRequired = status.Errorf(
		codes.FailedPrecondition,
		"A code to search for is required.",
	)
)

func (s *Server) ObjectTypeGet(
	ctx context.Context,
	req *pb.ObjectTypeGetRequest,
) (*pb.ObjectType, error) {
	if req.Code == "" {
		return nil, ErrCodeRequired
	}
	obj, err := s.store.ObjectTypeGet(req.Code)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve object type of %s: %s",
			req.Code,
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
	cur, err := s.store.ObjectTypeList(req.Any)
	if err != nil {
		return err
	}
	defer cur.Close()
	var msg pb.ObjectType
	for cur.Next() {
		if err = cur.Scan(&msg); err != nil {
			return err
		}
		if err = stream.Send(&msg); err != nil {
			return err
		}
	}
	return nil
}

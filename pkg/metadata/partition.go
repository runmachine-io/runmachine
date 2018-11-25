package metadata

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/proto"
)

// PartitionGet looks up a partition by UUID or name and returns a Partition
// protobuf message.
func (s *Server) PartitionGet(
	ctx context.Context,
	req *pb.PartitionGetRequest,
) (*pb.Partition, error) {
	if req.Filter == nil || req.Filter.Search == "" {
		return nil, ErrSearchRequired
	}
	obj, err := s.store.PartitionGet(req.Filter.Search)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve partition with UUID or name of %s: %s",
			req.Filter.Search,
			err,
		)
		return nil, ErrUnknown
	}
	return obj, nil
}

// PartitionList streams zero or more Partition objects back to the client that
// match a set of optional filters
func (s *Server) PartitionList(
	req *pb.PartitionListRequest,
	stream pb.RunmMetadata_PartitionListServer,
) error {
	cur, err := s.store.PartitionList(req.Any)
	if err != nil {
		return err
	}
	defer cur.Close()
	for cur.Next() {
		msg := &pb.Partition{}
		if err = cur.Scan(msg); err != nil {
			return err
		}
		if err = stream.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

package server

import (
	"context"
	"io"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/runmachine-io/runmachine/pkg/errors"
	metapb "github.com/runmachine-io/runmachine/proto"
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
	sess := req.Session
	metasess := &metapb.Session{
		User:      sess.User,
		Project:   sess.Project,
		Partition: sess.Partition,
	}
	metareq := &metapb.PartitionGetRequest{
		Session: metasess,
		Filter: &metapb.PartitionFilter{
			Search: req.Filter.Search,
		},
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	metaobj, err := mc.PartitionGet(context.Background(), metareq)
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
	return &pb.Partition{
		Uuid: metaobj.Uuid,
		Name: metaobj.Name,
	}, nil
}

// PartitionList streams zero or more Partition objects back to the client that
// match a set of optional filters
func (s *Server) PartitionList(
	req *pb.PartitionListRequest,
	stream pb.RunmAPI_PartitionListServer,
) error {
	sess := req.Session
	metasess := &metapb.Session{
		User:      sess.User,
		Project:   sess.Project,
		Partition: sess.Partition,
	}
	metareq := &metapb.PartitionListRequest{
		Session: metasess,
		// TODO(jaypipes): Any:     buildPartitionFilters(),
	}
	mc, err := s.metaClient()
	if err != nil {
		return err
	}
	metastream, err := mc.PartitionList(context.Background(), metareq)
	if err != nil {
		return err
	}

	objs := make([]*pb.Partition, 0)
	for {
		msg, err := metastream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		objs = append(
			objs, &pb.Partition{
				Uuid: msg.Uuid,
				Name: msg.Name,
			},
		)
	}
	for _, obj := range objs {
		if err = stream.Send(obj); err != nil {
			return err
		}
	}
	return nil
}

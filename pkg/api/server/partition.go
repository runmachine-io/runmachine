package server

import (
	"context"
	"io"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	metapb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/runmachine-io/runmachine/pkg/util"
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
	search := req.Filter.Search
	if util.IsUuidLike(search) {
		return s.metaPartitionGetByUuid(req.Session, search)
	} else {
		return s.metaPartitionGetByName(req.Session, search)
	}
}

// PartitionList streams zero or more Partition objects back to the client that
// match a set of optional filters
func (s *Server) PartitionList(
	req *pb.PartitionListRequest,
	stream pb.RunmAPI_PartitionListServer,
) error {
	metareq := &metapb.PartitionListRequest{
		Session: metaSession(req.Session),
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

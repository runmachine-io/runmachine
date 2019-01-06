package server

import (
	"context"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	yaml "gopkg.in/yaml.v2"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/runmachine-io/runmachine/pkg/api/types"
	metapb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
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
	return s.partitionGet(req.Session, req.Filter.Search)
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

// validatePartitionCreateRequest ensures that the data the user sent in the
// request payload can be unmarshal'd properly into YAML and contains all
// relevant fields
func (s *Server) validatePartitionCreateRequest(
	req *pb.CreateRequest,
) (*types.Partition, error) {
	var p types.Partition
	if err := yaml.Unmarshal(req.Payload, &p); err != nil {
		return nil, err
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Server) PartitionCreate(
	ctx context.Context,
	req *pb.CreateRequest,
) (*pb.PartitionCreateResponse, error) {
	// TODO(jaypipes): AUTHZ check if user can write a partition

	input, err := s.validatePartitionCreateRequest(req)
	if err != nil {
		return nil, err
	}

	// Save the partition in the metadata service
	partObj := &metapb.Partition{
		Uuid: input.Uuid,
		Name: input.Name,
	}
	created, err := s.partitionCreate(req.Session, partObj)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.AlreadyExists {
				return nil, ErrDuplicate
			}
		}
		s.log.ERR(
			"failed creating partition in metadata service: %s",
			err,
		)
		return nil, ErrUnknown
	}

	s.log.L1(
		"created new partition with UUID %s and name %s",
		created.Uuid,
		created.Name,
	)

	return &pb.PartitionCreateResponse{
		Partition: &pb.Partition{
			Uuid: created.Uuid,
			Name: created.Name,
		},
	}, nil
}

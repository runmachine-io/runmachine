package server

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

// PartitionGet looks up a partition by UUID or name and returns a Partition
// protobuf message.
func (s *Server) PartitionGet(
	ctx context.Context,
	req *pb.PartitionGetRequest,
) (*pb.Partition, error) {
	if req.Filter == nil {
		return nil, ErrSearchRequired
	} else {
		if req.Filter.UuidFilter == nil && req.Filter.NameFilter == nil {
			return nil, ErrSearchRequired
		}
	}
	obj, err := s.store.PartitionGet(req.Filter)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve partition with filter %s: %s",
			req.Filter, err,
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
	objs, err := s.store.PartitionList(req.Any)
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

// validatePartitionCreateRequest ensures that the data the user sent is valid and
// all referenced projects, partitions, and object types are correct.
func (s *Server) validatePartitionCreateRequest(
	req *pb.PartitionCreateRequest,
) (*pb.Partition, error) {
	part := req.Partition

	// Simple input data validations
	if part.Name == "" {
		return nil, ErrPartitionNameRequired
	}

	return part, nil
}

func (s *Server) PartitionCreate(
	ctx context.Context,
	req *pb.PartitionCreateRequest,
) (*pb.PartitionCreateResponse, error) {
	// TODO(jaypipes): AUTHZ check if user can write objects

	p, err := s.validatePartitionCreateRequest(req)
	if err != nil {
		return nil, err
	}
	changed, err := s.store.PartitionCreate(p)
	if err != nil {
		if err == errors.ErrDuplicate {
			return nil, ErrDuplicate
		}
		return nil, err
	}
	s.log.L1(
		"created new partition with UUID %s and name %s",
		changed.Uuid,
		changed.Name,
	)

	return &pb.PartitionCreateResponse{
		Partition: changed,
	}, nil
}

package server

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/runmachine-io/runmachine/pkg/util"
)

// PartitionGetByUuid looks up a partition by UUID and returns a Partition
// protobuf message. If no such partition was found, returns ErrNotFound.
func (s *Server) PartitionGetByUuid(
	ctx context.Context,
	req *pb.PartitionGetByUuidRequest,
) (*pb.Partition, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}
	uuid := req.Uuid
	if uuid == "" || !util.IsUuidLike(uuid) {
		return nil, ErrUuidRequired
	}
	obj, err := s.store.PartitionGetByUuid(util.NormalizeUuid(uuid))
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve partition with UUID '%s': %s",
			uuid, err,
		)
		return nil, ErrUnknown
	}
	return obj, nil
}

// PartitionGetByName looks up a partition by name and returns a Partition
// protobuf message. If no such partition was found, returns ErrNotFound.
func (s *Server) PartitionGetByName(
	ctx context.Context,
	req *pb.PartitionGetByNameRequest,
) (*pb.Partition, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}
	name := req.Name
	if name == "" {
		return nil, ErrNameRequired
	}
	obj, err := s.store.PartitionGetByName(name)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve partition with name '%s': %s",
			name, err,
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
	if err := s.checkSession(req.Session); err != nil {
		return err
	}
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

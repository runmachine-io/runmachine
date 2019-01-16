package server

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

// ObjectDefinitionGet looks up a object definition by partition, object type and
// object key and returns a ObjectDefinition protobuf message.
func (s *Server) ObjectDefinitionGet(
	ctx context.Context,
	req *pb.ObjectDefinitionGetRequest,
) (*pb.ObjectDefinition, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}

	// TODO(jaypipes): AUTHZ check user can read object definitions

	if req.Partition == "" {
		return nil, ErrPartitionRequired
	}
	if req.ObjectType == "" {
		return nil, ErrObjectTypeRequired
	}

	def, err := s.store.ObjectDefinitionGet(req.Partition, req.ObjectType)
	if err != nil {
		return nil, err
	}

	return def, nil
}

// validateObjectDefinitionSetRequest ensures that the data the user sent in
// the request is valid. It translates any partition name into a UUID and sets
// the ObjectDefinition.Partition to the partition's UUID if the Partition
// field was a name.
func (s *Server) validateObjectDefinitionSetRequest(
	req *pb.ObjectDefinitionSetRequest,
) error {
	def := req.ObjectDefinition

	// Validate the referred to type and partition actually exist
	// TODO(jaypipes): AUTHZ check user can specify partition
	part, err := s.store.PartitionGet(
		// Look up by UUID *or* name...
		&pb.PartitionFilter{
			UuidFilter: &pb.UuidFilter{
				Uuid: def.Partition,
			},
			NameFilter: &pb.NameFilter{
				Name: def.Partition,
			},
		},
	)
	if err != nil {
		if err == errors.ErrNotFound {
			return errPartitionNotFound(def.Partition)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed validating partition in object set: %s", err)
		return errors.ErrUnknown
	}
	def.Partition = part.Uuid

	objType, err := s.store.ObjectTypeGet(def.ObjectType)
	if err != nil {
		if err == errors.ErrNotFound {
			return errObjectTypeNotFound(def.ObjectType)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed validating object type in object set: %s", err)
		return errors.ErrUnknown
	}
	def.ObjectType = objType.Code
	return nil
}

// ObjectDefinitionSet receives an object definition to create or update and
// saves the object definition in backend storage
func (s *Server) ObjectDefinitionSet(
	ctx context.Context,
	req *pb.ObjectDefinitionSetRequest,
) (*pb.ObjectDefinitionSetResponse, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}

	// TODO(jaypipes): AUTHZ check for writing object definitions

	if err := s.validateObjectDefinitionSetRequest(req); err != nil {
		return nil, err
	}

	def := req.ObjectDefinition
	pk := def.Partition + ":" + def.ObjectType

	var existing *pb.ObjectDefinition
	existing, err := s.store.ObjectDefinitionGet(
		def.Partition, def.ObjectType,
	)
	if err != nil {
		if err != errors.ErrNotFound {
			s.log.ERR(
				"Failed trying to find existing object definition '%s': %s",
				pk,
				err,
			)
			// NOTE(jaypipes): don't return internal errors
			return nil, ErrUnknown
		}
	}
	if existing == nil {
		s.log.L3("creating new object definition '%s'...", pk)

		if err := s.store.ObjectDefinitionCreate(def); err != nil {
			return nil, err
		}
		s.log.L1("created new object definition '%s'", pk)
	} else {
		s.log.L3("updating object definition '%s'...", pk)
		// TODO(jaypipes): Update the object definition...
		s.log.L1("updated object definition '%s'", pk)
	}

	resp := &pb.ObjectDefinitionSetResponse{
		ObjectDefinition: def,
	}
	return resp, nil
}

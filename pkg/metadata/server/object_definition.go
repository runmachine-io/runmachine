package server

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

// ProviderDefinitionGet looks up either the global default object definition
// for providers or an object definition for providers in a specified partition
func (s *Server) ProviderDefinitionGet(
	ctx context.Context,
	req *pb.ProviderDefinitionGetRequest,
) (*pb.ObjectDefinition, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}

	// TODO(jaypipes): AUTHZ check user can read object definitions

	def, err := s.store.ObjectDefinitionGet(
		"runm.provider", req.Partition,
	)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return def, nil
}

// validateObjectDefinitionSetRequest ensures that the data the user sent in
// the request is valid. It translates any partition name into a UUID and sets
// the ObjectDefinition.Partition to the partition's UUID if the Partition
// field was a name.
func (s *Server) validateObjectDefinitionSetRequest(
	req *pb.ProviderDefinitionSetRequest,
) error {
	if req.Partition != "" {
		// Validate the referred to type and partition actually exist
		// TODO(jaypipes): AUTHZ check user can specify partition
		part, err := s.store.PartitionGet(
			&pb.PartitionFilter{
				UuidFilter: &pb.UuidFilter{
					Uuid: req.Partition,
				},
			},
		)
		if err != nil {
			if err == errors.ErrNotFound {
				return errPartitionNotFound(req.Partition)
			}
			// We don't want to leak internal implementation errors...
			s.log.ERR(
				"failed validating partition in object definition set: %s",
				err,
			)
			return errors.ErrUnknown
		}
		req.Partition = part.Uuid
	}

	return nil
}

// ProviderDefinitionSet receives an object definition to create or update and
// saves the object definition in backend storage
func (s *Server) ProviderDefinitionSet(
	ctx context.Context,
	req *pb.ProviderDefinitionSetRequest,
) (*pb.ObjectDefinitionSetResponse, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}

	// TODO(jaypipes): AUTHZ check for writing object definitions

	if err := s.validateObjectDefinitionSetRequest(req); err != nil {
		return nil, err
	}

	def := req.ObjectDefinition
	objType := "runm.provider"
	partUuid := req.Partition
	pk := objType + ":" + partUuid
	if partUuid == "" {
		pk += "default"
	}

	var existing *pb.ObjectDefinition
	existing, err := s.store.ObjectDefinitionGet(objType, partUuid)
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
	if err := s.store.ObjectDefinitionSet(objType, partUuid, def); err != nil {
		return nil, err
	}
	if existing == nil {
		s.log.L1("created new object definition '%s'", pk)
	} else {
		s.log.L1("updated object definition '%s'", pk)
	}

	resp := &pb.ObjectDefinitionSetResponse{
		ObjectDefinition: def,
	}
	return resp, nil
}

package server

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

// ProviderDefinitionGetGlobalDefault looks up the global default object
// definition for providers
func (s *Server) ProviderDefinitionGetGlobalDefault(
	ctx context.Context,
	req *pb.ProviderDefinitionGetGlobalDefaultRequest,
) (*pb.ObjectDefinition, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}

	def, err := s.store.ProviderDefinitionGet("", "")
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return def, nil
}

// ProviderDefinitionGetByPartition looks up object definition for providers in
// a specified partition. This object definition is used for providers when no
// definition override has been set for that particular provider type in the
// partition.
func (s *Server) ProviderDefinitionGetByPartition(
	ctx context.Context,
	req *pb.ProviderDefinitionGetByPartitionRequest,
) (*pb.ObjectDefinition, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}

	partUuid := req.PartitionUuid
	if partUuid == "" {
		return nil, ErrPartitionUuidRequired
	}
	// Validate the referred to partition actually exists
	// TODO(jaypipes): AUTHZ check user can specify partition
	_, err := s.store.PartitionGetByUuid(partUuid)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errPartitionNotFound(partUuid)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR(
			"failed validating partition in object definition set: %s",
			err,
		)
		return nil, errors.ErrUnknown
	}

	def, err := s.store.ProviderDefinitionGet(partUuid, "")
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return def, nil
}

// ProviderDefinitionGetByType looks up object definition for providers in
// having a specified provider type.
func (s *Server) ProviderDefinitionGetByType(
	ctx context.Context,
	req *pb.ProviderDefinitionGetByTypeRequest,
) (*pb.ObjectDefinition, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}

	provTypeCode := req.ProviderTypeCode
	if provTypeCode == "" {
		return nil, ErrProviderTypeCodeRequired
	}

	// Validate the referred to provider type actually exists

	_, err := s.store.ProviderTypeGetByCode(provTypeCode)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errProviderTypeNotFound(provTypeCode)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR(
			"failed validating provider type in object definition set: %s",
			err,
		)
		return nil, errors.ErrUnknown
	}

	def, err := s.store.ProviderDefinitionGet("", provTypeCode)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return def, nil
}

// ProviderDefinitionGetByPartitionAndType looks up object definition for
// providers in a specified partition and having a specified provider type.
func (s *Server) ProviderDefinitionGetByPartitionAndType(
	ctx context.Context,
	req *pb.ProviderDefinitionGetByPartitionAndTypeRequest,
) (*pb.ObjectDefinition, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}

	partUuid := req.PartitionUuid
	if partUuid == "" {
		return nil, ErrPartitionUuidRequired
	}

	provTypeCode := req.ProviderTypeCode
	if provTypeCode == "" {
		return nil, ErrProviderTypeCodeRequired
	}

	// Validate the referred to partition and provider type actually exists
	// TODO(jaypipes): AUTHZ check user can specify partition
	_, err := s.store.PartitionGetByUuid(partUuid)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errPartitionNotFound(partUuid)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR(
			"failed validating partition in object definition set: %s",
			err,
		)
		return nil, errors.ErrUnknown
	}

	_, err = s.store.ProviderTypeGetByCode(provTypeCode)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errProviderTypeNotFound(provTypeCode)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR(
			"failed validating provider type in object definition set: %s",
			err,
		)
		return nil, errors.ErrUnknown
	}

	def, err := s.store.ProviderDefinitionGet(partUuid, provTypeCode)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return def, nil
}

// validateProviderDefinitionSetRequest ensures that the data the user sent in
// the request is valid. It translates any partition name into a UUID and sets
// the ObjectDefinition.Partition to the partition's UUID if the Partition
// field was a name.
func (s *Server) validateProviderDefinitionSetRequest(
	req *pb.ProviderDefinitionSetRequest,
) error {
	partUuid := req.PartitionUuid
	if partUuid != "" {
		// Validate the referred to partition actually exists
		// TODO(jaypipes): AUTHZ check user can specify partition
		_, err := s.store.PartitionGetByUuid(partUuid)
		if err != nil {
			if err == errors.ErrNotFound {
				return errPartitionNotFound(partUuid)
			}
			// We don't want to leak internal implementation errors...
			s.log.ERR(
				"failed validating partition in object definition set: %s",
				err,
			)
			return errors.ErrUnknown
		}
	}
	provTypeCode := req.ProviderTypeCode
	if provTypeCode != "" {
		// Validate the referred to type actually exists
		// TODO(jaypipes): AUTHZ check user can specify provider type
		_, err := s.store.ProviderTypeGetByCode(provTypeCode)
		if err != nil {
			if err == errors.ErrNotFound {
				return errProviderTypeNotFound(provTypeCode)
			}
			// We don't want to leak internal implementation errors...
			s.log.ERR(
				"failed validating provider type in object definition set: %s",
				err,
			)
			return errors.ErrUnknown
		}
	}

	return nil
}

// ProviderDefinitionSet receives an object definition to create or update and
// saves the object definition in backend storage
func (s *Server) ProviderDefinitionSet(
	ctx context.Context,
	req *pb.ProviderDefinitionSetRequest,
) (*pb.ObjectDefinitionSetResponse, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}

	// TODO(jaypipes): AUTHZ check for writing object definitions

	if err := s.validateProviderDefinitionSetRequest(req); err != nil {
		return nil, err
	}

	def := req.ObjectDefinition
	objType := "runm.provider"
	partUuid := req.PartitionUuid
	provTypeCode := req.ProviderTypeCode
	pk := objType + ":" + partUuid
	if partUuid == "" {
		pk += "default"
	}
	if provTypeCode != "" {
		pk += ":" + provTypeCode
	}

	var existing *pb.ObjectDefinition
	existing, err := s.store.ProviderDefinitionGet(partUuid, provTypeCode)
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
	err = s.store.ProviderDefinitionSet(partUuid, provTypeCode, def)
	if err != nil {
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

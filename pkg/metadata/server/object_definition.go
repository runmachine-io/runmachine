package server

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
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
// the request's payload can be unmarshal'd properly into YAML, contains all
// relevant fields  and meets things like object definition validation
// checks.
//
// Returns a fully validated ObjectDefinitionWithReferences struct that
// describes the object definition and its related objects
func (s *Server) validateObjectDefinitionSetRequest(
	req *pb.ObjectDefinitionSetRequest,
) (*types.ObjectDefinitionWithReferences, error) {

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
			return nil, errPartitionNotFound(def.Partition)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed validating partition in object set: %s", err)
		return nil, errors.ErrUnknown
	}
	def.Partition = part.Uuid

	objType, err := s.store.ObjectTypeGet(def.ObjectType)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errObjectTypeNotFound(def.ObjectType)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed validating object type in object set: %s", err)
		return nil, errors.ErrUnknown
	}
	def.ObjectType = objType.Code

	// TODO(jaypipes): Validate if the user specified access permissions

	return &types.ObjectDefinitionWithReferences{
		Partition:  part,
		ObjectType: objType,
		Definition: def,
	}, nil
}

func (s *Server) ObjectDefinitionSet(
	ctx context.Context,
	req *pb.ObjectDefinitionSetRequest,
) (*pb.ObjectDefinitionSetResponse, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}

	// TODO(jaypipes): AUTHZ check for writing object definitions

	pdwr, err := s.validateObjectDefinitionSetRequest(req)
	if err != nil {
		return nil, err
	}

	def := pdwr.Definition

	// existing, err := s.store.ObjectDefinitionGet(p)
	//if err != nil {
	//	if err != errors.ErrNotFound {
	//		s.log.ERR(
	//			"Failed trying to find existing object definition '%s': %s",
	//			pk,
	//			err,
	//		)
	//		// NOTE(jaypipes): don't return internal errors
	//		return nil, ErrUnknown
	//	}
	//} else {
	//	def = existing
	//}
	var existing *pb.ObjectDefinition
	pk := pdwr.Partition.Uuid + ":" + pdwr.ObjectType.Code

	if existing == nil {
		s.log.L3("creating new object definition '%s'...", pk)

		// Set default access permissions to read/write by any role in the
		// creating project and read by anyone
		//if len(def.Permissions) == 0 {
		//	s.log.L3(
		//		"setting default permissions on object definition '%s' "+
		//			"to READ/WRITE for project '%s' and READ any",
		//		pk, req.Session.Project,
		//	)
		//	def.Permissions = []*pb.ObjectPermission{
		//		&pb.ObjectPermission{
		//			Project: &pb.StringValue{
		//				Value: req.Session.Project,
		//			},
		//			Permission: apitypes.PERMISSION_READ |
		//				apitypes.PERMISSION_WRITE,
		//		},
		//		&pb.ObjectPermission{
		//			Permission: apitypes.PERMISSION_READ,
		//		},
		//	}
		//} else {
		//	// Make sure that the project that created the object definition
		//	// can read and write it...
		//	foundProj := false
		//	for _, perm := range def.Permissions {
		//		if perm.Project != nil && perm.Project.Value == req.Session.Project {
		//			if (perm.Permission & apitypes.PERMISSION_WRITE) == 0 {
		//				s.log.L1(
		//					"added missing WRITE permission for "+
		//						"object definition '%s' in project '%s'",
		//					pk, perm.Project.Value,
		//				)
		//				perm.Permission |= apitypes.PERMISSION_WRITE
		//			}
		//			foundProj = true
		//		}
		//	}
		//	if !foundProj {
		//		s.log.L1(
		//			"added missing WRITE permission for object "+
		//				"definition '%s' in project '%s'",
		//			pk, req.Session.Project,
		//		)
		//		def.Permissions = append(
		//			def.Permissions,
		//			&pb.ObjectPermission{
		//				Project: &pb.StringValue{
		//					Value: req.Session.Project,
		//				},
		//				Permission: apitypes.PERMISSION_READ |
		//					apitypes.PERMISSION_WRITE,
		//			},
		//		)
		//	}
		//}

		if _, err := s.store.ObjectDefinitionCreate(pdwr); err != nil {
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

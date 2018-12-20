package metadata

import (
	"context"

	pb "github.com/runmachine-io/runmachine/proto"
	yaml "gopkg.in/yaml.v2"

	apitypes "github.com/runmachine-io/runmachine/pkg/api/types"
	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
)

// PropertyDefinitionDelete looks up one or more property definitions and
// deletes the definition from backend storage.
func (s *Server) PropertyDefinitionDelete(
	ctx context.Context,
	req *pb.PropertyDefinitionDeleteRequest,
) (*pb.PropertyDefinitionDeleteResponse, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}
	if len(req.Any) == 0 {
		return nil, ErrAtLeastOnePropertyDefinitionFilterRequired
	}

	filters, err := s.normalizePropertyDefinitionFilters(req.Session, req.Any)
	if err != nil {
		return nil, err
	}
	// Be extra-careful not to pass empty filters since that will delete all
	// objects...
	if len(filters) == 0 {
		return nil, ErrAtLeastOnePropertyDefinitionFilterRequired
	}

	pdwrs, err := s.store.PropertyDefinitionListWithReferences(filters)
	if err != nil {
		return nil, err
	}

	resErrors := make([]string, 0)
	numDeleted := uint64(0)
	for _, pdwr := range pdwrs {
		pk := &types.PropertyDefinitionPK{
			Partition:   pdwr.Partition.Uuid,
			ObjectType:  pdwr.Type.Code,
			PropertyKey: pdwr.Definition.Key,
		}
		s.log.L3("deleting property definition '%s'...", pk)
		if err = s.store.PropertyDefinitionDeleteByPK(pk); err != nil {
			resErrors = append(resErrors, err.Error())
		}
		// TODO(jaypipes): Send an event notification
		s.log.L1("deleted property definition '%s'", pk)
		numDeleted += 1
	}
	resp := &pb.PropertyDefinitionDeleteResponse{
		Errors:     resErrors,
		NumDeleted: numDeleted,
	}
	if len(resErrors) > 0 {
		return resp, ErrPropertyDefinitionDeleteFailed
	}
	return resp, nil
}

// PropertyDefinitionGet looks up a property definition by partition, object type and
// property key and returns a PropertyDefinition protobuf message.
func (s *Server) PropertyDefinitionGet(
	ctx context.Context,
	req *pb.PropertyDefinitionGetRequest,
) (*pb.PropertyDefinition, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}

	// TODO(jaypipes): AUTHZ check user can read property definitions

	if req.Filter == nil || req.Filter.Search == "" {
		return nil, ErrPropertyDefinitionFilterRequired
	}
	if req.Filter.Type == nil {
		return nil, ErrObjectTypeRequired
	}
	filters, err := s.expandPropertyDefinitionFilter(req.Session, req.Filter)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed retrieving property definition with search filter %s: %s",
			req.Filter.Search,
			err,
		)
		return nil, ErrUnknown
	}
	if len(filters) == 0 {
		return nil, ErrFailedExpandPropertyDefinitionFilters
	}

	objects, err := s.store.PropertyDefinitionList(filters)
	if err != nil {
		return nil, err
	}
	if len(objects) > 1 {
		return nil, ErrMultipleRecordsFound
	} else if len(objects) == 0 {
		return nil, ErrNotFound
	}

	return objects[0], nil
}

// PropertyDefinitionList streams PropertyDefinition protobuffer messages representing
// property definitions that matched the requested filters
func (s *Server) PropertyDefinitionList(
	req *pb.PropertyDefinitionListRequest,
	stream pb.RunmMetadata_PropertyDefinitionListServer,
) error {
	if err := checkSession(req.Session); err != nil {
		return err
	}

	filters, err := s.normalizePropertyDefinitionFilters(req.Session, req.Any)
	if err != nil {
		return err
	}

	objs, err := s.store.PropertyDefinitionList(filters)
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

// validatePropertyDefinitionSetRequest ensures that the data the user sent in
// the request's payload can be unmarshal'd properly into YAML, contains all
// relevant fields  and meets things like property definition validation
// checks.
//
// Returns a fully validated PropertyDefinitionWithReferences struct that
// describes the property definition and its related objects
func (s *Server) validatePropertyDefinitionSetRequest(
	req *pb.PropertyDefinitionSetRequest,
) (*types.PropertyDefinitionWithReferences, error) {
	// reads the supplied buffer which contains a YAML document describing the
	// property definition to create or update.
	def := &apitypes.PropertyDefinition{}
	if err := yaml.Unmarshal(req.Payload, def); err != nil {
		return nil, err
	}
	if err := def.Validate(); err != nil {
		return nil, errors.ErrInvalidPropertyDefinition(def.Type, def.Key, err)
	}

	// Validate the referred to type and partition actually exist
	// TODO(jaypipes): AUTHZ check user can specify partition
	part, err := s.store.PartitionGet(def.Partition)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errPartitionNotFound(def.Partition)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed validating partition in object set: %s", err)
		return nil, errors.ErrUnknown
	}

	objType, err := s.store.ObjectTypeGet(def.Type)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errObjectTypeNotFound(def.Type)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed validating object type in object set: %s", err)
		return nil, errors.ErrUnknown
	}

	// TODO(jaypipes): Validate if the user specified access permissions

	return &types.PropertyDefinitionWithReferences{
		Partition: part,
		Type:      objType,
		Definition: &pb.PropertyDefinition{
			Partition:   part.Uuid,
			Type:        objType.Code,
			Key:         def.Key,
			IsRequired:  def.Required,
			Permissions: types.APItoPBPropertyPermissions(def.Permissions),
			Schema:      types.APItoPBPropertySchema(def.Schema),
		},
	}, nil
}

func (s *Server) PropertyDefinitionSet(
	ctx context.Context,
	req *pb.PropertyDefinitionSetRequest,
) (*pb.PropertyDefinitionSetResponse, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}

	// TODO(jaypipes): AUTHZ check for writing property definitions

	pdwr, err := s.validatePropertyDefinitionSetRequest(req)
	if err != nil {
		return nil, err
	}

	def := pdwr.Definition
	pk := &types.PropertyDefinitionPK{
		Partition:   pdwr.Partition.Uuid,
		ObjectType:  pdwr.Type.Code,
		PropertyKey: def.Key,
	}

	existing, err := s.store.PropertyDefinitionGetByPK(pk)
	if err != nil {
		if err != errors.ErrNotFound {
			s.log.ERR(
				"Failed trying to find existing property definition '%s': %s",
				pk,
				err,
			)
			// NOTE(jaypipes): don't return internal errors
			return nil, ErrUnknown
		}
	} else {
		def = existing
	}

	if existing == nil {
		s.log.L3("creating new property definition '%s'...", pk)

		// Set default access permissions to read/write by any role in the
		// creating project and read by anyone
		if len(def.Permissions) == 0 {
			s.log.L3(
				"setting default permissions on property definition '%s' "+
					"to READ/WRITE for project '%s' and READ any",
				pk, req.Session.Project,
			)
			def.Permissions = []*pb.PropertyPermission{
				&pb.PropertyPermission{
					Project: &pb.StringValue{
						Value: req.Session.Project,
					},
					Permission: apitypes.PERMISSION_READ |
						apitypes.PERMISSION_WRITE,
				},
				&pb.PropertyPermission{
					Permission: apitypes.PERMISSION_READ,
				},
			}
		} else {
			// Make sure that the project that created the property definition
			// can read and write it...
			foundProj := false
			for _, perm := range def.Permissions {
				if perm.Project != nil && perm.Project.Value == req.Session.Project {
					if (perm.Permission & apitypes.PERMISSION_WRITE) == 0 {
						s.log.L1(
							"added missing WRITE permission for "+
								"property definition '%s' in project '%s'",
							pk, perm.Project.Value,
						)
						perm.Permission |= apitypes.PERMISSION_WRITE
					}
					foundProj = true
				}
			}
			if !foundProj {
				s.log.L1(
					"added missing WRITE permission for property "+
						"definition '%s' in project '%s'",
					pk, req.Session.Project,
				)
				def.Permissions = append(
					def.Permissions,
					&pb.PropertyPermission{
						Project: &pb.StringValue{
							Value: req.Session.Project,
						},
						Permission: apitypes.PERMISSION_READ |
							apitypes.PERMISSION_WRITE,
					},
				)
			}
		}

		if _, err := s.store.PropertyDefinitionCreate(pdwr); err != nil {
			return nil, err
		}
		s.log.L1("created new property definition '%s'", pk)
	} else {
		s.log.L3("updating property definition '%s'...", pk)
		// TODO(jaypipes): Update the property definition...
		s.log.L1("updated property definition '%s'", pk)
	}

	resp := &pb.PropertyDefinitionSetResponse{
		PropertyDefinition: def,
	}
	return resp, nil
}

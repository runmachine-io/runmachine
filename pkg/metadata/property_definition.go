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
		if err = s.store.PropertyDefinitionDelete(pdwr); err != nil {
			resErrors = append(resErrors, err.Error())
		}
		// TODO(jaypipes): Send an event notification
		s.log.L1(
			"user %s deleted property definition for property key %s of "+
				"object type %s in partition %s",
			req.Session.User,
			pdwr.Definition.Key,
			pdwr.Type.Code,
			pdwr.Partition.Uuid,
		)
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
			"failed to retrieve property definition with search filter %s: %s",
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
	if len(objects) != 1 {
		return nil, ErrMultipleRecordsFound
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

// validatePropertyDefinitionSetRequest ensures that the data the user sent in the
// request's payload can be unmarshal'd properly into YAML, contains all
// relevant fields.  and meets things like property definition validation checks.
//
// Returns a fully validated Object protobuffer message that is ready to send
// to backend storage.
func (s *Server) validatePropertyDefinitionSetRequest(
	req *pb.PropertyDefinitionSetRequest,
) (*types.PropertyDefinitionWithReferences, error) {
	// reads the supplied buffer which contains a YAML document describing the
	// property definition to create or update.
	obj := &apitypes.PropertyDefinition{}
	if err := yaml.Unmarshal(req.Payload, obj); err != nil {
		return nil, err
	}

	if obj.Type == "" {
		return nil, ErrObjectTypeRequired
	}
	if obj.Partition == "" {
		return nil, ErrPartitionRequired
	}
	if obj.Key == "" {
		return nil, ErrPropertyKeyRequired
	}
	if obj.Schema == nil {
		return nil, ErrSchemaRequired
	} else {
		if err := obj.Schema.Validate(); err != nil {
			return nil, errors.ErrInvalidPropertyDefinition(obj.Type, obj.Key, err)
		}
	}

	// Validate the referred to type, partition and project actually exist
	// TODO(jaypipes): AUTHZ check user can specify partition
	part, err := s.store.PartitionGet(obj.Partition)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errPartitionNotFound(obj.Partition)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed when validating partition in object set: %s", err)
		return nil, errors.ErrUnknown
	}

	objType, err := s.store.ObjectTypeGet(obj.Type)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errObjectTypeNotFound(obj.Type)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed when validating object type in object set: %s", err)
		return nil, errors.ErrUnknown
	}

	// TODO(jaypipes): Validate if the user specific access permissions

	return &types.PropertyDefinitionWithReferences{
		Partition: part,
		Type:      objType,
		Definition: &pb.PropertyDefinition{
			Partition:  part.Uuid,
			Type:       objType.Code,
			Key:        obj.Key,
			IsRequired: obj.Required,
			Schema:     obj.Schema.JSONSchemaString(),
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

	partition := pdwr.Partition.Uuid
	objType := pdwr.Type.Code
	def := pdwr.Definition
	propKey := def.Key

	existing, err := s.store.PropertyDefinitionGet(partition, objType, propKey)
	if err != nil {
		if err != errors.ErrNotFound {
			s.log.ERR(
				"Failed trying to find existing property definition "+
					"for %s:%s:%s: %s",
				partition,
				objType,
				propKey,
				err,
			)
			// NOTE(jaypipes): we don't return internal errors
			return nil, ErrUnknown
		}
	} else {
		def = existing
	}

	if existing == nil {
		s.log.L3(
			"creating new property definition %s:%s:%s...",
			partition,
			objType,
			propKey,
		)

		// Set default access permissions to read/write by any role in the
		// creating project
		if def.AccessPermissions == nil {
			def.AccessPermissions = []*pb.PropertyAccessPermission{
				&pb.PropertyAccessPermission{
					Project: &pb.StringValue{
						Value: req.Session.Project,
					},
					Permission: pb.AccessPermission_READ_WRITE,
				},
			}
		}

		// TODO(jaypipes): Make sure that the project that created the property
		// schema can read and write it

		if _, err := s.store.PropertyDefinitionCreate(pdwr); err != nil {
			return nil, err
		}
		s.log.L1(
			"created new property definition %s:%s:%s",
			partition,
			objType,
			propKey,
		)
	} else {
		s.log.L3(
			"updating property definition %s:%s:%s...",
			partition,
			objType,
			propKey,
		)
		// TODO(jaypipes): Update the property definition...
	}

	resp := &pb.PropertyDefinitionSetResponse{
		PropertyDefinition: def,
	}
	return resp, nil
}

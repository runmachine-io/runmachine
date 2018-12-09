package metadata

import (
	"context"

	pb "github.com/runmachine-io/runmachine/proto"
	yaml "gopkg.in/yaml.v2"

	apitypes "github.com/runmachine-io/runmachine/pkg/api/types"
	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
)

func (s *Server) PropertySchemaDelete(
	ctx context.Context,
	req *pb.PropertySchemaDeleteRequest,
) (*pb.PropertySchemaDeleteResponse, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}
	return nil, nil
}

// PropertySchemaGet looks up a property schema by partition, object type and
// property key and returns a PropertySchema protobuf message.
func (s *Server) PropertySchemaGet(
	ctx context.Context,
	req *pb.PropertySchemaGetRequest,
) (*pb.PropertySchema, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}

	// TODO(jaypipes): AUTHZ check user can read property schemas

	if req.Filter == nil || req.Filter.Search == "" {
		return nil, ErrPropertySchemaFilterRequired
	}

	if req.Filter.Type == nil {
		return nil, ErrObjectTypeRequired
	}
	// TODO(jaypipes): Look up whether object type exists

	var partSearch string
	if req.Filter.Partition != nil {
		partSearch = req.Filter.Partition.Search
	} else {
		// Use the session's partition if not specified
		partSearch = req.Session.Partition
	}
	if partSearch == "" {
		return nil, ErrPartitionRequired
	}
	part, err := s.store.PartitionGet(partSearch)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrPartitionUnknown
		}
		return nil, ErrUnknown
	}
	// TODO(jaypipes): AUTHZ check user can use partition

	obj, err := s.store.PropertySchemaGet(
		part.Uuid,
		req.Filter.Type.Search,
		req.Filter.Search,
	)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// Don't leak internal errors out
		return nil, ErrUnknown
	}
	return obj, nil
}

// PropertySchemaList streams PropertySchema protobuffer messages representing
// property schemas that matched the requested filters
func (s *Server) PropertySchemaList(
	req *pb.PropertySchemaListRequest,
	stream pb.RunmMetadata_PropertySchemaListServer,
) error {
	if err := checkSession(req.Session); err != nil {
		return err
	}

	filters, err := s.normalizePropertySchemaFilters(req.Session, req.Any)
	if err != nil {
		return err
	}

	objs, err := s.store.PropertySchemaList(filters)
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

// validatePropertySchemaSetRequest ensures that the data the user sent in the
// request's payload can be unmarshal'd properly into YAML, contains all
// relevant fields.  and meets things like property schema validation checks.
//
// Returns a fully validated Object protobuffer message that is ready to send
// to backend storage.
func (s *Server) validatePropertySchemaSetRequest(
	req *pb.PropertySchemaSetRequest,
) (*types.PropertySchemaWithReferences, error) {
	// reads the supplied buffer which contains a YAML document describing the
	// property schema to create or update.
	obj := &apitypes.PropertySchema{}
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
	if obj.Schema == "" {
		return nil, ErrSchemaRequired
	} else {
		// TODO(jaypipes): Validate the schema document provided
		s.log.L3("Validating property schema")
	}

	// Validate the referred to type, partition and project actually exist
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

	// TODO(jaypipes): AUTHZ check user can specify partition

	return &types.PropertySchemaWithReferences{
		Partition: part,
		Type:      objType,
		PropertySchema: &pb.PropertySchema{
			Partition: part.Uuid,
			Type:      objType.Code,
			Key:       obj.Key,
			Schema:    obj.Schema,
		},
	}, nil
}

func (s *Server) PropertySchemaSet(
	ctx context.Context,
	req *pb.PropertySchemaSetRequest,
) (*pb.PropertySchemaSetResponse, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}

	// TODO(jaypipes): AUTHZ check for writing property schemas

	pswr, err := s.validatePropertySchemaSetRequest(req)
	if err != nil {
		return nil, err
	}

	partition := pswr.Partition.Uuid
	objType := pswr.Type.Code
	obj := pswr.PropertySchema
	propKey := obj.Key

	existing, err := s.store.PropertySchemaGet(partition, objType, propKey)
	if err != nil {
		if err != errors.ErrNotFound {
			s.log.ERR(
				"Failed trying to find existing property schema for %s:%s:%s: %s",
				partition,
				objType,
				propKey,
				err,
			)
			// NOTE(jaypipes): we don't return internal errors
			return nil, ErrUnknown
		}
	}

	if existing == nil {
		s.log.L3("Creating new property schema %s:%s:%s", partition, objType, propKey)

		// Set default access permissions to read/write by any role in the
		// creating project
		if obj.AccessPermissions == nil {
			obj.AccessPermissions = []*pb.PropertyAccessPermission{
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

		if err := s.store.PropertySchemaCreate(obj); err != nil {
			return nil, err
		}
		resp := &pb.PropertySchemaSetResponse{
			PropertySchema: obj,
		}
		s.log.L1("Created new property schema %s:%s:%s", partition, objType, propKey)
		return resp, nil
	}

	s.log.L3("Updating property schema %s:%s:%s", partition, objType, propKey)

	// TODO(jaypipes): Update the property schema...

	return nil, nil
}

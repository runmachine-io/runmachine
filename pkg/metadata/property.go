package metadata

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/runmachine-io/runmachine/proto"

	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/storage"
)

var (
	ErrUnknown = status.Errorf(
		codes.Unknown,
		"an unknown error occurred.",
	)
	ErrNotFound = status.Errorf(
		codes.NotFound,
		"object could not be found.",
	)
	ErrPartitionUnknown = status.Errorf(
		codes.FailedPrecondition,
		"unknown partition.",
	)
	ErrPartitionRequired = status.Errorf(
		codes.FailedPrecondition,
		"partition is required.",
	)
	ErrObjectTypeRequired = status.Errorf(
		codes.FailedPrecondition,
		"object type is required.",
	)
	ErrPropertyKeyRequired = status.Errorf(
		codes.FailedPrecondition,
		"property key is required.",
	)
	ErrSchemaRequired = status.Errorf(
		codes.FailedPrecondition,
		"schema is required.",
	)
	ErrPropertySchemaObjectRequired = status.Errorf(
		codes.FailedPrecondition,
		"property schema object is required.",
	)
)

func (s *Server) PropertySchemaDelete(
	ctx context.Context,
	req *pb.PropertySchemaDeleteRequest,
) (*pb.PropertySchemaDeleteResponse, error) {
	return nil, nil
}

// PropertySchemaGet looks up a property schema by partition, object type and
// property key and returns a PropertySchema protobuf message.
func (s *Server) PropertySchemaGet(
	ctx context.Context,
	req *pb.PropertySchemaGetRequest,
) (*pb.PropertySchema, error) {
	// TODO(jaypipes): AUTHZ check user can read property schemas
	if req.ObjectType == "" {
		return nil, ErrObjectTypeRequired
	}
	// TODO(jaypipes): Look up whether object type exists

	var partSearch string
	if req.Partition == "" {
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

	if req.Key == "" {
		return nil, ErrPropertyKeyRequired
	}
	obj, err := s.store.PropertySchemaGet(
		part.Uuid,
		req.ObjectType,
		req.Key,
	)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, ErrUnknown
	}
	return obj, nil
}

func (s *Server) buildPropertySchemaFilter(
	filter *pb.PropertySchemaListFilter,
) (*storage.PropertySchemaFilter, error) {
	f := &storage.PropertySchemaFilter{}
	if filter.Partition != "" {
		// Verify that the partition exists and translate names to UUIDs
		part, err := s.store.PartitionGet(filter.Partition)
		if err != nil {
			return nil, err
		} else {
			f.PartitionUuid = part.Uuid
		}
	}
	return f, nil
}

// PropertySchemaList streams PropertySchema protobuffer messages representing
// property schemas that matched the requested filters
func (s *Server) PropertySchemaList(
	req *pb.PropertySchemaListRequest,
	stream pb.RunmMetadata_PropertySchemaListServer,
) error {
	any := make([]*storage.PropertySchemaFilter, 0)
	for _, filter := range req.Any {
		if f, err := s.buildPropertySchemaFilter(filter); err != nil {
			if err == errors.ErrNotFound {
				// Just return nil since clearly we can have no
				// property schemas matching an unknown partition
				return nil
			}
			return ErrUnknown
		} else if f != nil {
			any = append(any, f)
		}
	}
	if len(any) == 0 {
		// By default, filter by the session's partition
		part, err := s.store.PartitionGet(req.Session.Partition)
		if err != nil {
			if err == errors.ErrNotFound {
				// Just return nil since clearly we can have no
				// property schemas matching an unknown partition
				return nil
			}
			return ErrUnknown
		}
		any = append(
			any,
			&storage.PropertySchemaFilter{
				PartitionUuid: part.Uuid,
			},
		)
	}
	cur, err := s.store.PropertySchemaList(any)
	if err != nil {
		return err
	}
	defer cur.Close()
	var msg pb.PropertySchema
	for cur.Next() {
		if err = cur.Scan(&msg); err != nil {
			return err
		}
		if err = stream.Send(&msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) PropertySchemaSet(
	ctx context.Context,
	req *pb.PropertySchemaSetRequest,
) (*pb.PropertySchemaSetResponse, error) {
	// TODO(jaypipes): AUTHZ check for writing property schemas
	if req.PropertySchema == nil {
		return nil, ErrPropertySchemaObjectRequired
	}

	// First, validate the supplied property schema has the required fields.
	obj := req.PropertySchema

	if obj.Partition == "" {
		return nil, ErrPartitionRequired
	}

	if obj.ObjectType == "" {
		return nil, ErrObjectTypeRequired
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

	// TODO(jaypipes): AUTHZ check user can specify partition

	partition := obj.Partition
	objType := obj.ObjectType
	propKey := obj.Key

	existing, err := s.store.PropertySchemaGet(partition, objType, propKey)
	if err != nil {
		if err != ErrNotFound {
			s.log.ERR(
				"Failed trying to find existing property schema for %s:%s:%s: %s",
				partition,
				objType,
				propKey,
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

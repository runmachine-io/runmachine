package metadata

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/runmachine-io/runmachine/proto"

	"github.com/runmachine-io/runmachine/pkg/errors"
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
	if req.ObjectType == "" {
		return nil, ErrObjectTypeRequired
	}
	if req.Partition == "" {
		return nil, ErrPartitionRequired
	}
	if req.Key == "" {
		return nil, ErrPropertyKeyRequired
	}
	// TODO(jaypipes): AUTHZ check user can specify partition
	// TODO(jaypipes): AUTHZ check user can read property schemas
	obj, err := s.store.PropertySchemaGet(
		req.Partition,
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

func (s *Server) PropertySchemaList(
	req *pb.PropertySchemaListRequest,
	stream pb.RunmMetadata_PropertySchemaListServer,
) error {
	cur, err := s.store.PropertySchemaList(req)
	if err != nil {
		return err
	}
	defer cur.Close()
	var key string
	var msg pb.PropertySchema
	for cur.Next() {
		if err = cur.Scan(&key, &msg); err != nil {
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

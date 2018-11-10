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
	if req.ObjectType == nil {
		return nil, ErrObjectTypeRequired
	}
	version := uint32(1)
	if req.Version != nil {
		version = req.Version.Value
	}
	var partition string
	if req.Partition != nil {
		// TODO(jaypipes): AUTHZ check user can specify partition
		partition = req.Partition.Uuid
	} else {
		if req.Session.Partition == nil {
			return nil, ErrPartitionRequired
		}
		partition = req.Session.Partition.Uuid
	}
	// TODO(jaypipes): Validate the supplied object type even exists
	obj, err := s.store.PropertySchemaGet(
		partition,
		req.ObjectType.Code,
		req.Key,
		version,
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
	return nil, nil
}

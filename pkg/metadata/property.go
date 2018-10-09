package metadata

import (
	"context"

	pb "github.com/jaypipes/runmachine/proto"
)

func (s *Server) PropertySchemaDelete(
	ctx context.Context,
	req *pb.PropertySchemaDeleteRequest,
) (*pb.PropertySchemaDeleteResponse, error) {
	return nil, nil
}

func (s *Server) PropertySchemaGet(
	ctx context.Context,
	req *pb.PropertySchemaGetRequest,
) (*pb.PropertySchema, error) {
	return nil, nil
}

func (s *Server) PropertySchemaList(
	req *pb.PropertySchemaListRequest,
	stream pb.RunmMetadata_PropertySchemaListServer,
) error {
	return nil
}

func (s *Server) PropertySchemaSet(
	ctx context.Context,
	req *pb.PropertySchemaSetRequest,
) (*pb.PropertySchemaSetResponse, error) {
	return nil, nil
}

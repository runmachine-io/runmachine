package metadata

import (
	"context"

	pb "github.com/jaypipes/runmachine/proto"
)

func (s *Server) MetadataSchemaDelete(
	ctx context.Context,
	req *pb.MetadataSchemaDeleteRequest,
) (*pb.MetadataSchemaDeleteResponse, error) {
	return nil, nil
}

func (s *Server) MetadataSchemaGet(
	ctx context.Context,
	req *pb.MetadataSchemaGetRequest,
) (*pb.MetadataSchema, error) {
	return nil, nil
}

func (s *Server) MetadataSchemaList(
	req *pb.MetadataSchemaListRequest,
	stream pb.RunmMetadata_MetadataSchemaListServer,
) error {
	return nil
}

func (s *Server) MetadataSchemaSet(
	ctx context.Context,
	req *pb.MetadataSchemaSetRequest,
) (*pb.MetadataSchemaSetResponse, error) {
	return nil, nil
}

func (s *Server) ObjectDelete(
	ctx context.Context,
	req *pb.ObjectDeleteRequest,
) (*pb.ObjectDeleteResponse, error) {
	return nil, nil
}

func (s *Server) ObjectGet(
	ctx context.Context,
	req *pb.ObjectGetRequest,
) (*pb.Object, error) {
	return nil, nil
}

func (s *Server) ObjectList(
	req *pb.ObjectListRequest,
	stream pb.RunmMetadata_ObjectListServer,
) error {
	return nil
}

func (s *Server) ObjectSet(
	ctx context.Context,
	req *pb.ObjectSetRequest,
) (*pb.ObjectSetResponse, error) {
	return nil, nil
}

func (s *Server) ObjectMetadataItemList(
	req *pb.ObjectMetadataItemListRequest,
	stream pb.RunmMetadata_ObjectMetadataItemListServer,
) error {
	return nil
}

func (s *Server) ObjectMetadataSet(
	ctx context.Context,
	req *pb.ObjectMetadataSetRequest,
) (*pb.ObjectMetadataSetResponse, error) {
	return nil, nil
}

func (s *Server) ObjectTypeDelete(
	ctx context.Context,
	req *pb.ObjectTypeDeleteRequest,
) (*pb.ObjectTypeDeleteResponse, error) {
	return nil, nil
}

func (s *Server) ObjectTypeGet(
	ctx context.Context,
	req *pb.ObjectTypeGetRequest,
) (*pb.ObjectType, error) {
	return nil, nil
}

func (s *Server) ObjectTypeList(
	req *pb.ObjectTypeListRequest,
	stream pb.RunmMetadata_ObjectTypeListServer,
) error {
	return nil
}

func (s *Server) ObjectTypeSet(
	ctx context.Context,
	req *pb.ObjectTypeSetRequest,
) (*pb.ObjectTypeSetResponse, error) {
	return nil, nil
}

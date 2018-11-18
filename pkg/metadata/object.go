package metadata

import (
	"context"

	pb "github.com/runmachine-io/runmachine/proto"
)

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

func (s *Server) ObjectPropertiesList(
	req *pb.ObjectPropertiesListRequest,
	stream pb.RunmMetadata_ObjectPropertiesListServer,
) error {
	return nil
}

func (s *Server) ObjectPropertiesSet(
	ctx context.Context,
	req *pb.ObjectPropertiesSetRequest,
) (*pb.ObjectPropertiesSetResponse, error) {
	return nil, nil
}

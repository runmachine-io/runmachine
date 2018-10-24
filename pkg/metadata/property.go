package metadata

import (
	"context"

	pb "github.com/runmachine-io/runmachine/proto"
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

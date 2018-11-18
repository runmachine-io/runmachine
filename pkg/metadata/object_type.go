package metadata

import (
	"context"

	pb "github.com/runmachine-io/runmachine/proto"
)

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
	cur, err := s.store.ObjectTypeList(req.Any)
	if err != nil {
		return err
	}
	defer cur.Close()
	var key string
	var msg pb.ObjectType
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

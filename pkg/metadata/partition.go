package metadata

import (
	pb "github.com/runmachine-io/runmachine/proto"
)

// PartitionList streams zero or more Partition objects back to the client that
// match a set of optional filters
func (s *Server) PartitionList(
	req *pb.PartitionListRequest,
	stream pb.RunmMetadata_PartitionListServer,
) error {
	cur, err := s.store.PartitionList(req)
	if err != nil {
		return err
	}
	defer cur.Close()
	var key string
	var msg pb.Partition
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

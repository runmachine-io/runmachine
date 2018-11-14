package metadata

import (
	"context"

	"github.com/google/uuid"
	pb "github.com/runmachine-io/runmachine/proto"
)

func (s *Server) Bootstrap(
	ctx context.Context,
	req *pb.BootstrapRequest,
) (*pb.BootstrapResponse, error) {
	token := req.BootstrapToken
	partName := req.PartitionName
	var partUuid string
	if req.PartitionUuid == nil {
		partUuid = uuid.New().String()
	} else {
		partUuid = req.PartitionUuid.Value
	}

	partUuid = normalizeUuid(partUuid)

	if err := s.store.Bootstrap(token, partName, partUuid); err != nil {
		return nil, err
	}
	return &pb.BootstrapResponse{
		Partition: &pb.Partition{
			Name: partName,
			Uuid: partUuid,
		},
	}, nil
}

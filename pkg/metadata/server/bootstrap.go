package server

import (
	"context"

	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/runmachine-io/runmachine/pkg/util"
)

func (s *Server) Bootstrap(
	ctx context.Context,
	req *pb.BootstrapRequest,
) (*pb.BootstrapResponse, error) {
	token := req.BootstrapToken
	if token == "" {
		return nil, ErrBootstrapTokenRequired
	}
	partName := req.PartitionName
	if partName == "" {
		return nil, ErrPartitionNameRequired
	}

	var partUuid string
	if req.PartitionUuid == nil {
		partUuid = util.NewNormalizedUuid()
	} else {
		partUuid = util.NormalizeUuid(req.PartitionUuid.Value)
	}

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

package metadata

import (
	"context"

	"github.com/google/uuid"
	pb "github.com/runmachine-io/runmachine/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrBootstrapTokenRequired = status.Errorf(
		codes.FailedPrecondition,
		"bootstrap token is required.",
	)
	ErrPartitionNameRequired = status.Errorf(
		codes.FailedPrecondition,
		"partition name is required.",
	)
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

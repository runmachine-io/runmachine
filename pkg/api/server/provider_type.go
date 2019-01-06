package server

import (
	"context"
	"io"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	metapb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

// ProviderTypeList streams zero or more ProviderType objects back to the
// client that match a set of optional filters
func (s *Server) ProviderTypeList(
	req *pb.ProviderTypeListRequest,
	stream pb.RunmAPI_ProviderTypeListServer,
) error {
	metareq := &metapb.ProviderTypeListRequest{
		Session: metaSession(req.Session),
		// TODO(jaypipes): Any:     buildProviderTypeFilters(),
	}
	mc, err := s.metaClient()
	if err != nil {
		return err
	}
	metastream, err := mc.ProviderTypeList(context.Background(), metareq)
	if err != nil {
		return err
	}

	objs := make([]*pb.ProviderType, 0)
	for {
		msg, err := metastream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		objs = append(
			objs, &pb.ProviderType{
				Code:        msg.Code,
				Description: msg.Description,
			},
		)
	}
	for _, obj := range objs {
		if err = stream.Send(obj); err != nil {
			return err
		}
	}
	return nil
}

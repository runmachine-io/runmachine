package server

import (
	"context"
	"io"

	pb "github.com/runmachine-io/runmachine/proto"
)

// ProviderTypeGet looks up a provider type by code and returns a ProviderType
// protobuf message.
func (s *Server) ProviderTypeGet(
	ctx context.Context,
	req *pb.ProviderTypeGetRequest,
) (*pb.ProviderType, error) {
	if req.Filter == nil || req.Filter.Search == "" {
		return nil, ErrSearchRequired
	}
	return s.providerTypeGetByCode(req.Session, req.Filter.Search)
}

// ProviderTypeList streams zero or more ProviderType objects back to the
// client that match a set of optional filters
func (s *Server) ProviderTypeList(
	req *pb.ProviderTypeListRequest,
	stream pb.RunmAPI_ProviderTypeListServer,
) error {
	metareq := &pb.ProviderTypeFindRequest{
		Session: req.Session,
		// TODO(jaypipes): Any:     buildProviderTypeFilters(),
	}
	mc, err := s.metaClient()
	if err != nil {
		return err
	}
	metastream, err := mc.ProviderTypeFind(context.Background(), metareq)
	if err != nil {
		return err
	}

	for {
		msg, err := metastream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err = stream.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

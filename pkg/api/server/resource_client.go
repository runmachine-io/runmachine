package server

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/proto"
)

// TODO(jaypipes): Add retry behaviour
func (s *Server) resConnect(addr string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	// TODO(jaypipes): Don't hardcode this to WithInsecure
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// resClient returns a resource service client. We look up the resource service
// endpoint using the gsr service registry, connect to that endpoint, and if
// successful, return a constructed gRPC client to the resource service at that
// endpoint.
func (s *Server) resClient() (pb.RunmResourceClient, error) {
	if s.resclient != nil {
		return s.resclient, nil
	}
	// TODO(jaypipes): Move this code into a generic ServiceRegistry
	// struct/interface and allow for randomizing the pick of an endpoint from
	// multiple endpoints of the same service.
	var conn *grpc.ClientConn
	var addr string
	var err error
	for _, ep := range s.registry.Endpoints(s.cfg.ResourceServiceName) {
		addr = ep.Address
		s.log.L3("connecting to resource service at %s...", addr)
		if conn, err = s.resConnect(addr); err != nil {
			s.log.ERR(
				"failed to connect to resource service endpoint at %s: %s",
				addr, err,
			)
		} else {
			break
		}
	}
	if conn == nil {
		msg := "unable to connect to any resource service endpoint."
		s.log.ERR(msg)
		return nil, fmt.Errorf(msg)
	}
	s.resclient = pb.NewRunmResourceClient(conn)
	s.log.L2("connected to resource service at %s", addr)
	return s.resclient, nil
}

// providerCreate creates the supplied provider in the resource service. The
// data supplied has already been validated/checked. The supplied provider
// object may have fields updated on it when creation is successful (such as
// the provider's generation)
func (s *Server) providerCreate(
	sess *pb.Session,
	prov *pb.Provider,
) error {
	p := &pb.Provider{
		Uuid: prov.Uuid,
		Partition: &pb.Partition{
			Uuid: prov.Partition.Uuid,
		},
		ProviderType: &pb.ProviderType{
			Code: prov.ProviderType.Code,
		},
	}
	req := &pb.ProviderCreateRequest{
		Session:  sess,
		Provider: p,
	}
	rc, err := s.resClient()
	resp, err := rc.ProviderCreate(context.Background(), req)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.AlreadyExists {
				return errors.ErrDuplicate
			}
		}
		s.log.ERR(
			"failed saving provider with name '%s' in resource service: %s",
			prov.Name, err,
		)
		return errors.ErrUnknown
	}
	prov.Generation = resp.Provider.Generation
	return nil
}

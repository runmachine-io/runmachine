package server

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/runmachine-io/runmachine/pkg/api/types"
	"github.com/runmachine-io/runmachine/pkg/errors"
	respb "github.com/runmachine-io/runmachine/pkg/resource/proto"
)

// metaSession transforms an API protobuffer Session message into a metadata
// service protobuffer Session message
func resSession(sess *pb.Session) *respb.Session {
	return &respb.Session{
		User:      sess.User,
		Project:   sess.Project,
		Partition: sess.Partition,
	}
}

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
func (s *Server) resClient() (respb.RunmResourceClient, error) {
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
	s.resclient = respb.NewRunmResourceClient(conn)
	s.log.L2("connected to resource service at %s", addr)
	return s.resclient, nil
}

// providerCreate creates the supplied provider in the resource service. The
// data supplied has already been validated/checked.
func (s *Server) providerCreate(
	sess *pb.Session,
	prov *types.Provider,
) (*respb.Provider, error) {
	p := &respb.Provider{
		Uuid:         prov.Uuid,
		Partition:    prov.Partition,
		ProviderType: prov.ProviderType,
	}
	req := &respb.ProviderCreateRequest{
		Session:  resSession(sess),
		Provider: p,
	}
	rc, err := s.resClient()
	resp, err := rc.ProviderCreate(context.Background(), req)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.AlreadyExists {
				return nil, errors.ErrDuplicate
			}
		}
		s.log.ERR(
			"failed saving provider with name '%s' in resource service: %s",
			prov.Name, err,
		)
		return nil, errors.ErrUnknown
	}
	return resp.Provider, nil
}

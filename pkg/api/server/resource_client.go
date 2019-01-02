package server

import (
	"context"
	"fmt"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/runmachine-io/runmachine/pkg/errors"
	respb "github.com/runmachine-io/runmachine/pkg/resource/proto"
	"google.golang.org/grpc"
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

// providerGetByUuid returns a provider matching the supplied UUID key. If no
// such provider could be found, returns (nil, ErrNotFound)
func (s *Server) providerGetByUuid(
	sess *pb.Session,
	uuid string,
) (*pb.Provider, error) {
	req := &respb.ProviderGetRequest{
		Session: resSession(sess),
		Uuid:    uuid,
	}
	rc, err := s.resClient()
	if err != nil {
		return nil, err
	}
	rec, err := rc.ProviderGet(context.Background(), req)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve provider with UUID %s: %s",
			uuid, err,
		)
		return nil, ErrUnknown
	}
	return &pb.Provider{
		Partition:    rec.Partition,
		ProviderType: rec.ProviderType.String(),
		Uuid:         rec.Uuid,
		Generation:   rec.Generation,
	}, nil
}

package server

import (
	"fmt"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	metapb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"google.golang.org/grpc"
)

// metaSession transforms an API protobuffer Session message into a metadata
// service protobuffer Session message
func metaSession(sess *pb.Session) *metapb.Session {
	return &metapb.Session{
		User:      sess.User,
		Project:   sess.Project,
		Partition: sess.Partition,
	}
}

// TODO(jaypipes): Add retry behaviour
func (s *Server) metaConnect(addr string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	// TODO(jaypipes): Don't hardcode this to WithInsecure
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// metaClient returns a metadata service client. We look up the metadata
// service endpoint using the gsr service registry, connect to that endpoint,
// and if successful, return a constructed gRPC client to the metadata service
// at that endpoint.
func (s *Server) metaClient() (metapb.RunmMetadataClient, error) {
	if s.metaclient != nil {
		return s.metaclient, nil
	}
	var conn *grpc.ClientConn
	var addr string
	var err error
	for _, ep := range s.registry.Endpoints(s.cfg.MetadataServiceName) {
		addr = ep.Address
		s.log.L3("connecting to metadata service at %s...", addr)
		if conn, err = s.metaConnect(addr); err != nil {
			s.log.ERR(
				"failed to connect to metadata service endpoint at %s: %s",
				addr, err,
			)
		} else {
			break
		}
	}
	if conn == nil {
		msg := "unable to connect to any metadata service endpoint."
		s.log.ERR(msg)
		return nil, fmt.Errorf(msg)
	}
	s.metaclient = metapb.NewRunmMetadataClient(conn)
	s.log.L2("connected to metadata service at %s", addr)
	return s.metaclient, nil
}

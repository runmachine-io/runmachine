package server

import (
	"context"
	"fmt"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/runmachine-io/runmachine/pkg/errors"
	metapb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/runmachine-io/runmachine/pkg/util"
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
	// TODO(jaypipes): Move this code into a generic ServiceRegistry
	// struct/interface and allow for randomizing the pick of an endpoint from
	// multiple endpoints of the same service.
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

// partitionGet returns a partition record matching the supplied UUID or name
// If no such partition could be found, returns (nil, ErrNotFound)
func (s *Server) partitionGet(
	sess *pb.Session,
	search string,
) (*pb.Partition, error) {
	if util.IsUuidLike(search) {
		return s.partitionGetByUuid(sess, search)
	}
	return s.partitionGetByName(sess, search)
}

// partitionGetByUuid returns a partition record matching the supplied UUID
// key. If no such partition could be found, returns (nil, ErrNotFound)
func (s *Server) partitionGetByUuid(
	sess *pb.Session,
	uuid string,
) (*pb.Partition, error) {
	req := &metapb.PartitionGetRequest{
		Session: metaSession(sess),
		Filter: &metapb.PartitionFilter{
			UuidFilter: &metapb.UuidFilter{
				Uuid:      uuid,
				UsePrefix: false,
			},
		},
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	rec, err := mc.PartitionGet(context.Background(), req)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve partition with UUID %s: %s",
			uuid, err,
		)
		return nil, ErrUnknown
	}
	return &pb.Partition{
		Uuid: rec.Uuid,
		Name: rec.Name,
	}, nil
}

// partitionGetByName returns a partition record matching the supplied name.
// If no such partition could be found, returns (nil, ErrNotFound)
func (s *Server) partitionGetByName(
	sess *pb.Session,
	name string,
) (*pb.Partition, error) {
	req := &metapb.PartitionGetRequest{
		Session: metaSession(sess),
		Filter: &metapb.PartitionFilter{
			NameFilter: &metapb.NameFilter{
				Name:      name,
				UsePrefix: false,
			},
		},
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	rec, err := mc.PartitionGet(context.Background(), req)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve partition with name %s: %s",
			name, err,
		)
		return nil, ErrUnknown
	}
	return &pb.Partition{
		Uuid: rec.Uuid,
		Name: rec.Name,
	}, nil
}

// objectGetUuid returns a UUID matching the supplied object type and name. If
// no such object could be found, returns ("", ErrNotFound)
func (s *Server) uuidFromName(
	sess *pb.Session,
	objType string,
	name string,
) (string, error) {
	req := &metapb.ObjectGetRequest{
		Session: metaSession(sess),
		Filter: &metapb.ObjectFilter{
			ObjectType: &metapb.ObjectTypeFilter{
				Search:    objType,
				UsePrefix: false,
			},
			Name:      name,
			UsePrefix: false,
		},
	}
	mc, err := s.metaClient()
	if err != nil {
		return "", err
	}
	rec, err := mc.ObjectGet(context.Background(), req)
	if err != nil {
		if err == errors.ErrNotFound {
			return "", ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve object of type %s with name %s: %s",
			objType, name, err,
		)
		return "", ErrUnknown
	}
	return rec.Uuid, nil
}

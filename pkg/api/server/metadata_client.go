package server

import (
	"context"
	"fmt"
	"io"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/runmachine-io/runmachine/pkg/util"
	metapb "github.com/runmachine-io/runmachine/proto"
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

// partitionsGetMatching takes an API PartitionFilter and returns a list of
// Partition messages matching the filter
func (s *Server) partitionsGetMatchingFilter(
	sess *pb.Session,
	filter *pb.SearchFilter,
) ([]*metapb.Partition, error) {
	mfil := &metapb.PartitionFilter{}
	if util.IsUuidLike(filter.Search) {
		mfil.UuidFilter = &metapb.UuidFilter{
			Uuid: filter.Search,
		}
	} else {
		mfil.NameFilter = &metapb.NameFilter{
			Name:      filter.Search,
			UsePrefix: filter.UsePrefix,
		}
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	req := &metapb.PartitionListRequest{
		Session: metaSession(sess),
		Any:     []*metapb.PartitionFilter{mfil},
	}
	stream, err := mc.PartitionList(context.Background(), req)
	if err != nil {
		return nil, err
	}

	msgs := make([]*metapb.Partition, 0)
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
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
	req := &metapb.PartitionGetByUuidRequest{
		Session: metaSession(sess),
		Uuid:    uuid,
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	rec, err := mc.PartitionGetByUuid(context.Background(), req)
	if err != nil {
		return nil, err
	}
	// TODO(jaypipes): Use a single proto namespace so we don't always need to
	// copy data like this...
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
	req := &metapb.PartitionGetByNameRequest{
		Session: metaSession(sess),
		Name:    name,
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	rec, err := mc.PartitionGetByName(context.Background(), req)
	if err != nil {
		return nil, err
	}
	// TODO(jaypipes): Use a single proto namespace so we don't always need to
	// copy data like this...
	return &pb.Partition{
		Uuid: rec.Uuid,
		Name: rec.Name,
	}, nil
}

// partitionCreate takes a new partition definition and returns a
// metapb.Partition message representing the newly-created partition in the
// metadata service.
func (s *Server) partitionCreate(
	sess *pb.Session,
	part *metapb.Partition,
) (*metapb.Partition, error) {
	req := &metapb.PartitionCreateRequest{
		Session:   metaSession(sess),
		Partition: part,
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	resp, err := mc.PartitionCreate(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.Partition, nil
}

// uuidFromName returns a UUID matching the supplied object type and name. If
// no such object could be found, returns ("", ErrNotFound)
func (s *Server) uuidFromName(
	sess *pb.Session,
	objType string,
	name string,
) (string, error) {
	req := &metapb.ObjectGetByNameRequest{
		Session:        metaSession(sess),
		ObjectTypeCode: objType,
		Name:           name,
	}
	mc, err := s.metaClient()
	if err != nil {
		return "", err
	}
	rec, err := mc.ObjectGetByName(context.Background(), req)
	if err != nil {
		return "", err
	}
	return rec.Uuid, nil
}

// objectFromUuid returns an Object message matching the supplied object UUID.
// If no such object could be found, returns ("", ErrNotFound)
func (s *Server) objectFromUuid(
	sess *pb.Session,
	uuid string,
) (*metapb.Object, error) {
	req := &metapb.ObjectGetByUuidRequest{
		Session: metaSession(sess),
		Uuid:    uuid,
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	rec, err := mc.ObjectGetByUuid(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// nameFromUuid returns a name matching the supplied object UUID. If no such
// object could be found, returns ("", ErrNotFound)
func (s *Server) nameFromUuid(
	sess *pb.Session,
	uuid string,
) (string, error) {
	obj, err := s.objectFromUuid(sess, uuid)
	if err != nil {
		return "", err
	}
	return obj.Name, nil
}

// providerTypeGetByCode returns a provider type record matching the supplied
// code. If no such provider type could be found, returns (nil, ErrNotFound)
func (s *Server) providerTypeGetByCode(
	sess *pb.Session,
	code string,
) (*pb.ProviderType, error) {
	req := &metapb.ProviderTypeGetByCodeRequest{
		Session: metaSession(sess),
		Code:    code,
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	rec, err := mc.ProviderTypeGetByCode(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return &pb.ProviderType{
		Code:        rec.Code,
		Description: rec.Description,
	}, nil
}

// objectCreate creates a supplied object in the metadata service. The supplied
// pointer to an Object is updated with fields from the newly-created object in
// the metadata service, including any auto-created UUIDs
func (s *Server) objectCreate(
	sess *pb.Session,
	obj *metapb.Object,
) error {
	req := &metapb.ObjectCreateRequest{
		Session: metaSession(sess),
		Object:  obj,
	}
	mc, err := s.metaClient()
	if err != nil {
		return err
	}
	resp, err := mc.ObjectCreate(context.Background(), req)
	if err != nil {
		return err
	}
	// Make sure that our object's UUID is set to the (possibly auto-created)
	// UUID returned by the metadata service
	obj.Uuid = resp.Object.Uuid
	return nil
}

// objectDelete deletes any object with one of the supplied UUIDs from the
// metadata service
func (s *Server) objectDelete(
	sess *pb.Session,
	uuids []string,
) error {
	req := &metapb.ObjectDeleteByUuidsRequest{
		Session: metaSession(sess),
		Uuids:   uuids,
	}
	mc, err := s.metaClient()
	if err != nil {
		return err
	}
	_, err = mc.ObjectDeleteByUuids(context.Background(), req)
	if err != nil {
		return err
	}
	return nil
}

// objectsGetMatching takes a slice of pointers to object filters and returns
// matching metapb.Object messages
func (s *Server) objectsGetMatching(
	sess *pb.Session,
	any []*metapb.ObjectFilter,
) ([]*metapb.Object, error) {
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	req := &metapb.ObjectListRequest{
		Session: metaSession(sess),
		Any:     any,
	}
	stream, err := mc.ObjectList(context.Background(), req)
	if err != nil {
		return nil, err
	}

	msgs := make([]*metapb.Object, 0)
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

package metadata

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/storage"
	pb "github.com/runmachine-io/runmachine/proto"
)

func (s *Server) ObjectDelete(
	ctx context.Context,
	req *pb.ObjectDeleteRequest,
) (*pb.ObjectDeleteResponse, error) {
	return nil, nil
}

func (s *Server) ObjectGet(
	ctx context.Context,
	req *pb.ObjectGetRequest,
) (*pb.Object, error) {
	return nil, nil
}

// buildPartitionObjectFilters is used to expand an ObjectFilter, which may
// contain PartitionFilter and ObjectTypeFilter objects that themselves may
// resolve to multiple partitions or object types, to a set of
// PartitionObjectFilter objects. A PartitionObjectFilter is used to describe a
// filter on objects in a *specific* partition and having a *specific* object
// type.
func (s *Server) buildPartitionObjectFilters(
	filter *pb.ObjectFilter,
) ([]*storage.PartitionObjectFilter, error) {
	res := make([]*storage.PartitionObjectFilter, 0)
	// A set of partition UUIDs that we'll create PartitionObjectFilters with.
	// These are the UUIDs of any partitions that match the PartitionFilter in
	// the supplied pb.ObjectFilter
	partUuids := make(map[string]bool, 0)
	// A set of object type codes that we'll create PartitionObjectFilters
	// with. These are the codes of object types that match the
	// ObjectTypeFilter in the supplied ObjectFilter
	otCodes := make(map[string]bool, 0)

	if filter.Partition != nil {
		// Verify that the requested partition(s) exist(s) and for each
		// requested partition match, construct a new PartitionObjectFilter
		cur, err := s.store.PartitionList([]*pb.PartitionFilter{filter.Partition})
		if err != nil {
			return nil, err
		}
		defer cur.Close()

		var part pb.Partition
		for cur.Next() {
			if err = cur.Scan(&part); err != nil {
				return nil, err
			}
			partUuids[part.Uuid] = true
		}
	}
	if filter.ObjectType != nil {
		// Verify that the object type even exists
		cur, err := s.store.ObjectTypeList([]*pb.ObjectTypeFilter{filter.ObjectType})
		if err != nil {
			return nil, err
		}
		defer cur.Close()

		var ot pb.ObjectType
		for cur.Next() {
			if err = cur.Scan(&ot); err != nil {
				return nil, err
			}
			otCodes[ot.Code] = true
		}
	}

	for partUuid := range partUuids {
		if len(otCodes) == 0 {
			f := &storage.PartitionObjectFilter{
				PartitionUuid: partUuid,
			}
			res = append(res, f)
		} else {
			for otCode := range otCodes {
				f := &storage.PartitionObjectFilter{
					PartitionUuid:  partUuid,
					ObjectTypeCode: otCode,
				}
				res = append(res, f)
			}
		}
	}

	// Now that we've expanded our partitions and object types, add in the
	// original ObjectFilter's Search and UsePrefix for each
	// PartitionObjectFilter we've created
	for _, pf := range res {
		pf.Search = filter.Search
		pf.UsePrefix = filter.UsePrefix
	}
	return res, nil
}

func (s *Server) ObjectList(
	req *pb.ObjectListRequest,
	stream pb.RunmMetadata_ObjectListServer,
) error {
	any := make([]*storage.PartitionObjectFilter, 0)
	for _, filter := range req.Any {
		if pfs, err := s.buildPartitionObjectFilters(filter); err != nil {
			if err == errors.ErrNotFound {
				// Just continue since clearly we can have no objects matching
				// an unknown partition but we need to OR together all filters,
				// which is why we don't just return nil here
				continue
			}
			return ErrUnknown
		} else if len(pfs) > 0 {
			for _, pf := range pfs {
				any = append(any, pf)
			}
		}
	}
	if len(any) == 0 {
		// By default, filter by the session's partition
		part, err := s.store.PartitionGet(req.Session.Partition)
		if err != nil {
			if err == errors.ErrNotFound {
				// Just return nil since clearly we can have no
				// property schemas matching an unknown partition
				return nil
			}
			return ErrUnknown
		}
		any = append(
			any,
			&storage.PartitionObjectFilter{
				PartitionUuid: part.Uuid,
			},
		)
	}
	cur, err := s.store.ObjectList(any)
	if err != nil {
		return err
	}
	defer cur.Close()
	var msg pb.Object
	for cur.Next() {
		if err = cur.Scan(&msg); err != nil {
			return err
		}
		if err = stream.Send(&msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) ObjectSet(
	ctx context.Context,
	req *pb.ObjectSetRequest,
) (*pb.ObjectSetResponse, error) {
	return nil, nil
}

func (s *Server) ObjectPropertiesList(
	req *pb.ObjectPropertiesListRequest,
	stream pb.RunmMetadata_ObjectPropertiesListServer,
) error {
	return nil
}

func (s *Server) ObjectPropertiesSet(
	ctx context.Context,
	req *pb.ObjectPropertiesSetRequest,
) (*pb.ObjectPropertiesSetResponse, error) {
	return nil, nil
}

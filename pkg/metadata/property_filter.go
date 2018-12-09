package metadata

import (
	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
	pb "github.com/runmachine-io/runmachine/proto"
)

// defaultPropertySchemaFilter returns the default property schema filter if
// the user didn't specify any filters themselves. This filter will be across
// the partition that the user's session is on.
func (s *Server) defaultPropertySchemaFilter(
	session *pb.Session,
) (*types.PropertySchemaFilter, error) {
	part, err := s.store.PartitionGet(session.Partition)
	if err != nil {
		if err == errors.ErrNotFound {
			// Just return nil since clearly we can have no
			// property schemas matching an unknown partition
			s.log.L3(
				"'%s' listed property schemas with no filters "+
					"and supplied unknown partition '%s' in the session",
				session.User,
				session.Partition,
			)
		}
		return nil, err
	}
	return &types.PropertySchemaFilter{
		Partition: part,
	}, nil
}

// expandPropertySchemaFilter is used to expand an PropertySchemaFilter, which may contain
// PartitionFilter and ObjectTypeFilter objects that themselves may resolve to
// multiple partitions or object types, to a set of types.PropertySchemaFilter
// objects. A types.PropertySchemaFilter is used to describe a filter on objects in
// a *specific* partition and having a *specific* object type.
func (s *Server) expandPropertySchemaFilter(
	session *pb.Session,
	filter *pb.PropertySchemaFilter,
) ([]*types.PropertySchemaFilter, error) {
	res := make([]*types.PropertySchemaFilter, 0)
	// A set of partition UUIDs that we'll create types.PropertySchemaFilters with.
	// These are the UUIDs of any partitions that match the PartitionFilter in
	// the supplied pb.PropertySchemaFilter
	partitions := make([]*pb.Partition, 0)
	// A set of object type codes that we'll create types.PropertySchemaFilters
	// with. These are the codes of object types that match the
	// ObjectTypeFilter in the supplied PropertySchemaFilter
	objTypes := make([]*pb.ObjectType, 0)

	if filter.Partition != nil {
		// Verify that the requested partition(s) exist(s) and for each
		// requested partition match, construct a new types.PropertySchemaFilter
		cur, err := s.store.PartitionList([]*pb.PartitionFilter{filter.Partition})
		if err != nil {
			return nil, err
		}
		defer cur.Close()

		for cur.Next() {
			part := &pb.Partition{}
			if err = cur.Scan(part); err != nil {
				return nil, err
			}
			partitions = append(partitions, part)
		}
		if len(partitions) == 0 {
			return nil, errors.ErrNotFound
		}
	} else {
		// By default, filter by the session's partition if the user didn't
		// specify any filtering.
		part, err := s.store.PartitionGet(session.Partition)
		if err != nil {
			if err == errors.ErrNotFound {
				// Just return nil since clearly we can have no
				// property schemas matching an unknown partition
				s.log.L3(
					"'%s' listed objects with no filters "+
						"and supplied unknown partition '%s' in the session",
					session.User,
					session.Partition,
				)
			}
			return nil, err
		}
		partitions = append(partitions, part)
	}

	if filter.Type != nil {
		// Verify that the object type even exists
		objTypes, err := s.store.ObjectTypeList([]*pb.ObjectTypeFilter{filter.Type})
		if err != nil {
			return nil, err
		}
		if len(objTypes) == 0 {
			return nil, errors.ErrNotFound
		}
	}

	// OK, if we've expanded partition or object type, we need to construct
	// types.PropertySchemaFilter objects containing the combination of all the
	// expanded partitions and object types.
	if len(partitions) > 0 {
		for _, p := range partitions {
			if len(objTypes) == 0 {
				f := &types.PropertySchemaFilter{
					Partition: p,
				}
				res = append(res, f)
			} else {
				for _, ot := range objTypes {
					f := &types.PropertySchemaFilter{
						Partition: p,
						Type:      ot,
					}
					res = append(res, f)
				}
			}
		}
	} else if len(objTypes) > 0 {
		for _, ot := range objTypes {
			f := &types.PropertySchemaFilter{
				Type: ot,
			}
			res = append(res, f)
		}
	}

	// If we've expanded the supplied partition filters into multiple
	// types.PropertySchemaFilters, then we need to add our supplied
	// PropertySchemaFilter's search and use prefix for the property key.  If
	// we supplied no partition filters, then go ahead and just return a single
	// types.PropertySchemaFilter with the search term and prefix indicator for
	// the property key.
	if filter.Search != "" {
		if len(res) > 0 {
			// Now that we've expanded our partitions and object types, add in the
			// original PropertySchemaFilter's Search and UsePrefix for each
			// types.PropertySchemaFilter we've created
			for _, pf := range res {
				pf.Search = filter.Search
				pf.UsePrefix = filter.UsePrefix
			}
		} else {
			res = append(
				res,
				&types.PropertySchemaFilter{
					Search:    filter.Search,
					UsePrefix: filter.UsePrefix,
				},
			)
		}
	}
	return res, nil
}

// normalizePropertySchemaFilters is passed a Session object and a slice of
// PropertySchemaFilter protobuffer messages. It then expands those supplied
// PropertySchemaFilter messages if they contain partition or object type
// filters that have a prefix. If no PropertySchemaFilter messages are passed
// to this method, it returns the default types.PropertySchemaFilter which will
// return all property schemas for the Session's partition
func (s *Server) normalizePropertySchemaFilters(
	session *pb.Session,
	any []*pb.PropertySchemaFilter,
) ([]*types.PropertySchemaFilter, error) {
	res := make([]*types.PropertySchemaFilter, 0)
	for _, filter := range any {
		if pfs, err := s.expandPropertySchemaFilter(session, filter); err != nil {
			if err == errors.ErrNotFound {
				// Just continue since clearly we can have no property schemas
				// matching an unknown partition but we need to OR together all
				// filters, which is why we don't just return nil here
				continue
			}
			s.log.ERR(
				"normalizePropertySchemaFilters: failed to expand property "+
					"schema filter %s: %s",
				filter,
				err,
			)
			return nil, errors.ErrUnknown
		} else if len(pfs) > 0 {
			for _, pf := range pfs {
				res = append(res, pf)
			}
		}
	}

	if len(res) == 0 {
		if len(any) == 0 {
			// At least one filter should have been expanded
			defFilter, err := s.defaultPropertySchemaFilter(session)
			if err != nil {
				return nil, ErrFailedExpandPropertySchemaFilters
			}
			res = append(res, defFilter)
		} else {
			// The user asked for object types that don't exist, partitions
			// that don't exist, etc.
			return nil, nil
		}
	}
	return res, nil
}

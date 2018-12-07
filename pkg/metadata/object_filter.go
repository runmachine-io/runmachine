package metadata

import (
	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/storage"
	pb "github.com/runmachine-io/runmachine/proto"
)

// defaultObjectFilter returns the default partition object filter if the user
// didn't specify any filters themselves. This filter will be across the
// partition that the user's session is on.
func (s *Server) defaultObjectFilter(
	session *pb.Session,
) (*storage.ObjectListFilter, error) {
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
	return &storage.ObjectListFilter{
		Partition: part,
		Project:   session.Project,
	}, nil
}

// expandObjectFilter is used to expand an ObjectFilter, which may contain
// PartitionFilter and ObjectTypeFilter objects that themselves may resolve to
// multiple partitions or object types, to a set of ObjectListFilter
// objects. A ObjectListFilter is used to describe a filter on objects in
// a *specific* partition and having a *specific* object type.
func (s *Server) expandObjectFilter(
	session *pb.Session,
	filter *pb.ObjectFilter,
) ([]*storage.ObjectListFilter, error) {
	res := make([]*storage.ObjectListFilter, 0)
	// A set of partition UUIDs that we'll create ObjectListFilters with.
	// These are the UUIDs of any partitions that match the PartitionFilter in
	// the supplied pb.ObjectFilter
	partitions := make([]*pb.Partition, 0)
	// A set of object type codes that we'll create ObjectListFilters
	// with. These are the codes of object types that match the
	// ObjectTypeFilter in the supplied ObjectFilter
	objTypes := make([]*pb.ObjectType, 0)

	if filter.Partition != nil {
		// Verify that the requested partition(s) exist(s) and for each
		// requested partition match, construct a new ObjectListFilter
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

	if filter.ObjectType != nil {
		// Verify that the object type even exists
		cur, err := s.store.ObjectTypeList([]*pb.ObjectTypeFilter{filter.ObjectType})
		if err != nil {
			return nil, err
		}
		defer cur.Close()

		for cur.Next() {
			ot := &pb.ObjectType{}
			if err = cur.Scan(ot); err != nil {
				return nil, err
			}
			objTypes = append(objTypes, ot)
		}
		if len(objTypes) == 0 {
			return nil, errors.ErrNotFound
		}
	}

	// Default the object list to filtering by the session's project if the
	// user didn't specify a specific project to filter on
	if filter.Project == "" {
		filter.Project = session.Project
	} else {
		// TODO(jaypipes): Determine if the user has the ability to list
		// objects in other projects...
	}

	// OK, if we've expanded partition or object type, we need to construct
	// ObjectListFilter objects containing the combination of all the
	// expanded partitions and object types.
	if len(partitions) > 0 {
		for _, p := range partitions {
			if len(objTypes) == 0 {
				f := &storage.ObjectListFilter{
					Partition: p,
				}
				res = append(res, f)
			} else {
				for _, ot := range objTypes {
					f := &storage.ObjectListFilter{
						Partition:  p,
						ObjectType: ot,
					}
					res = append(res, f)
				}
			}
		}
	} else if len(objTypes) > 0 {
		for _, ot := range objTypes {
			f := &storage.ObjectListFilter{
				ObjectType: ot,
			}
			res = append(res, f)
		}
	}

	// If we've expanded the supplied partition filters into multiple
	// ObjectListFilters, then we need to add our supplied ObjectFilter's
	// search and use prefix for the object's UUID/name. If we supplied no
	// partition filters, then go ahead and just return a single
	// ObjectListFilter with the search term and prefix indicator for the
	// object.
	if filter.Search != "" || filter.Project != "" {
		if len(res) > 0 {
			// Now that we've expanded our partitions and object types, add in the
			// original ObjectFilter's Search and UsePrefix for each
			// ObjectListFilter we've created
			for _, pf := range res {
				pf.Project = filter.Project
				pf.Search = filter.Search
				pf.UsePrefix = filter.UsePrefix
			}
		} else {
			res = append(
				res,
				&storage.ObjectListFilter{
					Project:   filter.Project,
					Search:    filter.Search,
					UsePrefix: filter.UsePrefix,
				},
			)
		}
	}
	return res, nil
}

// normalizeObjectFilters is passed a Session object and a slice of
// ObjectFilter messages. It then expands those supplied ObjectFilter messages
// if they contain partition or object type filters that have a prefix. If no
// ObjectFilter messages are passed to this method, it returns the default
// ObjectListFilter which will return all objects for the Session's partition
// and project.
func (s *Server) normalizeObjectFilters(
	session *pb.Session,
	any []*pb.ObjectFilter,
) ([]*storage.ObjectListFilter, error) {
	res := make([]*storage.ObjectListFilter, 0)
	for _, filter := range any {
		if pfs, err := s.expandObjectFilter(session, filter); err != nil {
			if err == errors.ErrNotFound {
				// Just continue since clearly we can have no objects matching
				// an unknown partition but we need to OR together all filters,
				// which is why we don't just return nil here
				continue
			}
			s.log.ERR("normalizeObjectFilters: failed to expand object filter %s: %s", filter, err)
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
			defFilter, err := s.defaultObjectFilter(session)
			if err != nil {
				return nil, ErrFailedExpandObjectFilters
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
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
) (*storage.PartitionObjectFilter, error) {
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
	return &storage.PartitionObjectFilter{
		Partition: part,
	}, nil
}

// expandObjectFilter is used to expand an ObjectFilter, which may contain
// PartitionFilter and ObjectTypeFilter objects that themselves may resolve to
// multiple partitions or object types, to a set of PartitionObjectFilter
// objects. A PartitionObjectFilter is used to describe a filter on objects in
// a *specific* partition and having a *specific* object type.
func (s *Server) expandObjectFilter(
	session *pb.Session,
	filter *pb.ObjectFilter,
) ([]*storage.PartitionObjectFilter, error) {
	res := make([]*storage.PartitionObjectFilter, 0)
	// A set of partition UUIDs that we'll create PartitionObjectFilters with.
	// These are the UUIDs of any partitions that match the PartitionFilter in
	// the supplied pb.ObjectFilter
	partitions := make([]*pb.Partition, 0)
	// A set of object type codes that we'll create PartitionObjectFilters
	// with. These are the codes of object types that match the
	// ObjectTypeFilter in the supplied ObjectFilter
	objTypes := make([]*pb.ObjectType, 0)

	if filter.Partition != nil {
		// Verify that the requested partition(s) exist(s) and for each
		// requested partition match, construct a new PartitionObjectFilter
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

	// OK, if we've expanded partition or object type, we need to construct
	// PartitionObjectFilter objects containing the combination of all the
	// expanded partitions and object types.
	if len(partitions) > 0 {
		for _, p := range partitions {
			if len(objTypes) == 0 {
				f := &storage.PartitionObjectFilter{
					Partition: p,
				}
				res = append(res, f)
			} else {
				for _, ot := range objTypes {
					f := &storage.PartitionObjectFilter{
						Partition:  p,
						ObjectType: ot,
					}
					res = append(res, f)
				}
			}
		}
	} else if len(objTypes) > 0 {
		for _, ot := range objTypes {
			f := &storage.PartitionObjectFilter{
				ObjectType: ot,
			}
			res = append(res, f)
		}
	}

	// If we've expanded the supplied partition filters into multiple
	// PartitionObjectFilters, then we need to add our supplied ObjectFilter's
	// search and use prefix for the object's UUID/name. If we supplied no
	// partition filters, then go ahead and just return a single
	// PartitionObjectFilter with the search term and prefix indicator for the
	// object.
	if filter.Search != "" || filter.Project != "" {
		if len(res) > 0 {
			// Now that we've expanded our partitions and object types, add in the
			// original ObjectFilter's Search and UsePrefix for each
			// PartitionObjectFilter we've created
			for _, pf := range res {
				pf.Project = filter.Project
				pf.Search = filter.Search
				pf.UsePrefix = filter.UsePrefix
			}
		} else {
			res = append(
				res,
				&storage.PartitionObjectFilter{
					Project:   filter.Project,
					Search:    filter.Search,
					UsePrefix: filter.UsePrefix,
				},
			)
		}
	}
	return res, nil
}

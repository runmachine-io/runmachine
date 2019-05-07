package server

import (
	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/conditions"
	pb "github.com/runmachine-io/runmachine/proto"
)

// defaultObjectFilter returns the default partition object filter if the user
// didn't specify any filters themselves. This filter will be across the
// partition that the user's session is on.
func (s *Server) defaultObjectFilter(
	session *pb.Session,
) (*conditions.ObjectCondition, error) {
	p, err := s.store.PartitionGetByUuid(session.Partition)
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
	return &conditions.ObjectCondition{
		PartitionCondition: conditions.PartitionEqual(p.Uuid),
		ProjectCondition:   session.Project,
	}, nil
}

// expandObjectFilter is used to expand an ObjectFilter, which may contain
// PartitionFilter and ObjectTypeFilter objects that themselves may resolve to
// multiple partitions or object types, to a set of types.ObjectCondition
// objects. A types.ObjectCondition is used to describe a filter on objects in
// a *specific* partition and having a *specific* object type.
func (s *Server) expandObjectFilter(
	session *pb.Session,
	filter *pb.ObjectFilter,
) ([]*conditions.ObjectCondition, error) {
	res := make([]*conditions.ObjectCondition, 0)
	var err error
	// A set of partition UUIDs that we'll create conditions.ObjectConditions with.
	// These are the UUIDs of any partitions that match the PartitionFilter in
	// the supplied pb.ObjectFilter
	var partitions []*pb.Partition
	// A set of object type codes that we'll create conditions.ObjectConditions
	// with. These are the codes of object types that match the
	// ObjectTypeFilter in the supplied ObjectFilter
	var objTypes []*pb.ObjectType

	if filter.PartitionFilter != nil {
		// Verify that the requested partition(s) exist(s)
		numParts := len(filter.PartitionFilter.Uuids)
		pfils := make([]*pb.PartitionFindFilter, numParts)
		for x, partUuid := range filter.PartitionFilter.Uuids {
			pfils[x] = &pb.PartitionFindFilter{
				UuidFilter: &pb.UuidFilter{
					Uuid: partUuid,
				},
			}
		}
		partitions, err = s.store.PartitionFind(pfils)
		if err != nil {
			return nil, err
		}
		if len(partitions) == 0 {
			return nil, errors.ErrNotFound
		}
	} else {
		// By default, filter by the session's partition if the user didn't
		// specify any filtering.
		part, err := s.store.PartitionGetByUuid(session.Partition)
		if err != nil {
			if err == errors.ErrNotFound {
				// Just return nil since clearly we can have no
				// property schemas matching an unknown partition
				s.log.L3(
					"'%s' listed objects with no partition filters "+
						"and supplied unknown partition '%s' in the session",
					session.User,
					session.Partition,
				)
			}
			return nil, err
		}
		partitions = append(partitions, part)
	}

	if filter.ObjectTypeFilter != nil {
		// Verify that the object type even exists
		objTypes, err = s.store.ObjectTypeFind(
			[]*pb.ObjectTypeFilter{filter.ObjectTypeFilter},
		)
		if err != nil {
			return nil, err
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
	// conditions.ObjectCondition objects containing the combination of all the
	// expanded partitions and object types.
	if len(partitions) > 0 {
		for _, p := range partitions {
			if len(objTypes) == 0 {
				f := &conditions.ObjectCondition{
					PartitionCondition: conditions.PartitionEqual(p.Uuid),
				}
				res = append(res, f)
			} else {
				for _, ot := range objTypes {
					f := &conditions.ObjectCondition{
						PartitionCondition:  conditions.PartitionEqual(p.Uuid),
						ObjectTypeCondition: conditions.ObjectTypeEqual(ot.Code),
					}
					res = append(res, f)
				}
			}
		}
	} else if len(objTypes) > 0 {
		for _, ot := range objTypes {
			f := &conditions.ObjectCondition{
				ObjectTypeCondition: conditions.ObjectTypeEqual(ot.Code),
			}
			res = append(res, f)
		}
	}

	// If we've expanded the supplied partition filters into multiple
	// types.ObjectConditions, then we need to add our supplied ObjectFilter's
	// search and use prefix for the object's UUID/name. If we supplied no
	// partition filters, then go ahead and just return a single
	// types.ObjectCondition with the search term and prefix indicator for the
	// object.
	if filter.NameFilter != nil || filter.UuidFilter != nil || filter.Project != "" || filter.PropertyFilter != nil {
		if len(res) == 0 {
			res = append(res, &conditions.ObjectCondition{})
		}
		// Now that we've expanded our partitions and object type, add in the
		// original ObjectFilter's Search and UsePrefix for each
		// conditions.ObjectCondition we've created
		for _, pf := range res {
			if filter.UuidFilter != nil {
				pf.UuidCondition = conditions.UuidEqual(filter.UuidFilter.Uuid)
			}
			if filter.NameFilter != nil {
				if filter.NameFilter.UsePrefix {
					pf.NameCondition = conditions.NameLike(
						filter.NameFilter.Name,
					)
				} else {
					pf.NameCondition = conditions.NameEqual(
						filter.NameFilter.Name,
					)
				}
			}
			if filter.PropertyFilter != nil {
				pf.PropertyCondition = conditions.PropertyConditionFromFilter(
					filter.PropertyFilter,
				)
			}
			pf.ProjectCondition = filter.Project
		}
	}
	return res, nil
}

// normalizeObjectFilters is passed a Session object and a slice of
// ObjectFilter messages. It then expands those supplied ObjectFilter messages
// if they contain partition or object type filters that have a prefix. If no
// ObjectFilter messages are passed to this method, it returns the default
// types.ObjectCondition which will return all objects for the Session's partition
// and project.
func (s *Server) normalizeObjectFilters(
	session *pb.Session,
	any []*pb.ObjectFilter,
) ([]*conditions.ObjectCondition, error) {
	res := make([]*conditions.ObjectCondition, 0)
	for _, filter := range any {
		if pfs, err := s.expandObjectFilter(session, filter); err != nil {
			if err == errors.ErrNotFound {
				// Just continue since clearly we can have no objects matching
				// an unknown partition but we need to OR together all filters,
				// which is why we don't just return nil here
				continue
			}
			s.log.ERR(
				"normalizeObjectFilters: failed to expand object filter %s: %s",
				filter, err,
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
			defFilter, err := s.defaultObjectFilter(session)
			if err != nil {
				return nil, ErrFailedExpandObjectFilters
			}
			res = append(res, defFilter)
		} else {
			// The user asked for object types that don't exist, partitions
			// that don't exist, etc.
			return nil, ErrFailedExpandObjectFilters
		}
	}
	return res, nil
}

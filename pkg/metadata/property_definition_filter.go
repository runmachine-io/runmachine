package metadata

import (
	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
	pb "github.com/runmachine-io/runmachine/proto"
)

// defaultPropertyDefinitionFilter returns the default property definition filter if
// the user didn't specify any filters themselves. This filter will be across
// the partition that the user's session is on.
func (s *Server) defaultPropertyDefinitionFilter(
	session *pb.Session,
) (*types.PropertyDefinitionCondition, error) {
	p, err := s.store.PartitionGet(session.Partition)
	if err != nil {
		if err == errors.ErrNotFound {
			// Just return nil since clearly we can have no
			// property definitions matching an unknown partition
			s.log.L3(
				"'%s' listed property definitions with no filters "+
					"and supplied unknown partition '%s' in the session",
				session.User,
				session.Partition,
			)
		}
		return nil, err
	}
	return &types.PropertyDefinitionCondition{
		PartitionCondition: &types.PartitionCondition{
			Op:        types.OP_EQUAL,
			Partition: p,
		},
	}, nil
}

// expandPropertyDefinitionFilter is used to expand an
// PropertyDefinitionFilter, which may contain PartitionFilter and
// ObjectTypeFilter objects that themselves may resolve to multiple partitions
// or object types, to a set of types.PropertyDefinitionCondition objects. A
// types.PropertyDefinitionCondition is used to describe a filter on objects in a
// *specific* partition and having a *specific* object type.
func (s *Server) expandPropertyDefinitionFilter(
	session *pb.Session,
	filter *pb.PropertyDefinitionFilter,
) ([]*types.PropertyDefinitionCondition, error) {
	res := make([]*types.PropertyDefinitionCondition, 0)
	var err error
	// A set of partition UUIDs that we'll create
	// types.PropertyDefinitionConditions with.  These are the UUIDs of any
	// partitions that match the PartitionFilter in the supplied
	// pb.PropertyDefinitionFilter
	var partitions []*pb.Partition
	// A set of object type codes that we'll create
	// types.PropertyDefinitionConditions with. These are the codes of object
	// types that match the ObjectTypeFilter in the supplied
	// PropertyDefinitionFilter
	var objTypes []*pb.ObjectType

	if filter.Partition != nil {
		// Verify that the requested partition(s) exist(s)
		partitions, err = s.store.PartitionList(
			[]*pb.PartitionFilter{filter.Partition},
		)
		if err != nil {
			return nil, err
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
				// property definitions matching an unknown partition
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
		objTypes, err = s.store.ObjectTypeList(
			[]*pb.ObjectTypeFilter{filter.ObjectType},
		)
		if err != nil {
			return nil, err
		}
		if len(objTypes) == 0 {
			return nil, errors.ErrNotFound
		}
	}

	// OK, if we've expanded partition or object type, we need to construct
	// types.PropertyDefinitionCondition objects containing the combination of all the
	// expanded partitions and object types.
	if len(partitions) > 0 {
		for _, p := range partitions {
			if len(objTypes) == 0 {
				f := &types.PropertyDefinitionCondition{
					PartitionCondition: &types.PartitionCondition{
						Op:        types.OP_EQUAL,
						Partition: p,
					},
				}
				res = append(res, f)
			} else {
				for _, ot := range objTypes {
					f := &types.PropertyDefinitionCondition{
						PartitionCondition: &types.PartitionCondition{
							Op:        types.OP_EQUAL,
							Partition: p,
						},
						ObjectTypeCondition: &types.ObjectTypeCondition{
							Op:         types.OP_EQUAL,
							ObjectType: ot,
						},
					}
					res = append(res, f)
				}
			}
		}
	} else if len(objTypes) > 0 {
		for _, ot := range objTypes {
			f := &types.PropertyDefinitionCondition{
				ObjectTypeCondition: &types.ObjectTypeCondition{
					Op:         types.OP_EQUAL,
					ObjectType: ot,
				},
			}
			res = append(res, f)
		}
	}

	// If we've expanded the supplied partition filters into multiple
	// types.PropertyDefinitionConditions, then we need to add our supplied
	// PropertyDefinitionFilter's search and use prefix for the property key.  If
	// we supplied no partition filters, then go ahead and just return a single
	// types.PropertyDefinitionCondition with the search term and prefix indicator for
	// the property key.
	if filter.Key != "" || filter.Uuid != "" {
		if len(res) == 0 {
			res = append(res, &types.PropertyDefinitionCondition{})
		}
		// Now that we've expanded our partitions and object types, add in the
		// original PropertyDefinitionFilter's Search and UsePrefix for each
		// types.PropertyDefinitionCondition we've created
		for _, pf := range res {
			if filter.Key != "" {
				op := types.OP_EQUAL
				if filter.UsePrefix {
					op = types.OP_GREATER_THAN_EQUAL
				}
				pf.PropertyKeyCondition = &types.PropertyKeyCondition{
					Op:          op,
					PropertyKey: filter.Key,
				}
			}
			if filter.Uuid != "" {
				pf.UuidCondition = &types.UuidCondition{
					Op:   types.OP_EQUAL,
					Uuid: filter.Uuid,
				}
			}
		}
	}
	return res, nil
}

// normalizePropertyDefinitionFilters is passed a Session object and a slice of
// PropertyDefinitionFilter protobuffer messages. It then expands those supplied
// PropertyDefinitionFilter messages if they contain partition or object type
// filters that have a prefix. If no PropertyDefinitionFilter messages are passed
// to this method, it returns the default types.PropertyDefinitionCondition which will
// return all property definitions for the Session's partition
func (s *Server) normalizePropertyDefinitionFilters(
	session *pb.Session,
	any []*pb.PropertyDefinitionFilter,
) ([]*types.PropertyDefinitionCondition, error) {
	res := make([]*types.PropertyDefinitionCondition, 0)
	for _, filter := range any {
		if pfs, err := s.expandPropertyDefinitionFilter(session, filter); err != nil {
			if err == errors.ErrNotFound {
				// Just continue since clearly we can have no property definitions
				// matching an unknown partition but we need to OR together all
				// filters, which is why we don't just return nil here
				continue
			}
			s.log.ERR(
				"normalizePropertyDefinitionFilters: failed to expand property "+
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
			defFilter, err := s.defaultPropertyDefinitionFilter(session)
			if err != nil {
				return nil, ErrFailedExpandPropertyDefinitionFilters
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

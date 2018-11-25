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
		nParts := 0
		for cur.Next() {
			if err = cur.Scan(&part); err != nil {
				return nil, err
			}
			partUuids[part.Uuid] = true
			nParts += 1
		}
		if nParts == 0 {
			return nil, errors.ErrNotFound
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
		nTypes := 0
		for cur.Next() {
			if err = cur.Scan(&ot); err != nil {
				return nil, err
			}
			otCodes[ot.Code] = true
			nTypes += 1
		}
		if nTypes == 0 {
			return nil, errors.ErrNotFound
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
			return errors.ErrUnknown
		} else if len(pfs) > 0 {
			for _, pf := range pfs {
				any = append(any, pf)
			}
		}
	}

	if len(any) == 0 {
		if len(req.Any) > 0 {
			// If the user specified filters but due to specifying unknown
			// partitions or object types, there were no non-empty filters
			// produced, we return nil to indicate no records were found
			s.log.L3(
				"object_list: no partition object filters created from %s",
				req.Any,
			)
			return nil
		}
		// By default, filter by the session's partition if the user didn't
		// specify any filtering.
		part, err := s.store.PartitionGet(req.Session.Partition)
		if err != nil {
			if err == errors.ErrNotFound {
				// Just return nil since clearly we can have no
				// property schemas matching an unknown partition
				return nil
			}
			return errors.ErrUnknown
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

// validateObjectSetRequest ensures that the data the user sent in the
// request's Before and After elements makes sense and meets things like
// property schema validation checks. Returns the object type for the new
// object.
func (s *Server) validateObjectSetRequest(
	req *pb.ObjectSetRequest,
) (*pb.ObjectType, error) {
	after := req.After

	// Simple input data validations
	if after.ObjectType == "" {
		return nil, ErrObjectTypeRequired
	}
	if after.Partition == "" {
		return nil, ErrPartitionRequired
	}

	// Validate the referred to type, partition and project actually exist
	p, err := s.store.PartitionGet(after.Partition)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errPartitionNotFound(after.Partition)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed when validating partition in object set: %s", err)
		return nil, errors.ErrUnknown
	}
	after.Partition = p.Uuid

	ot, err := s.store.ObjectTypeGet(after.ObjectType)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errObjectTypeNotFound(after.ObjectType)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed when validating object type in object set: %s", err)
		return nil, errors.ErrUnknown
	}
	after.ObjectType = ot.Code

	if req.Before == nil {
		// TODO(jaypipes): User expects to create a new object with the after
		// image. Ensure we don't have an existing object with the supplied
		// UUID, or if UUID is empty (indicating the user wants the UUID to be
		// auto-created), no existing object with the supplied name exists in
		// the partition or project scope.
		switch ot.Scope {
		case pb.ObjectTypeScope_PROJECT:
		case pb.ObjectTypeScope_PARTITION:
		}
	}

	// TODO(jaypipes): property schema validation checks
	return ot, nil
}

func (s *Server) ObjectSet(
	ctx context.Context,
	req *pb.ObjectSetRequest,
) (*pb.ObjectSetResponse, error) {
	// TODO(jaypipes): AUTHZ check if user can write objects

	ot, err := s.validateObjectSetRequest(req)
	if err != nil {
		return nil, err
	}

	var changed *pb.Object
	if req.Before == nil {
		s.log.L3(
			"creating new object of type %s in partition %s with name %s...",
			req.After.ObjectType,
			req.After.Partition,
			req.After.Name,
		)
		changed, err = s.store.ObjectCreate(req.After, ot)
		if err != nil {
			return nil, err
		}
		s.log.L1(
			"created new object with UUID %s of type %s in partition %s with name %s",
			changed.Uuid,
			req.After.ObjectType,
			req.After.Partition,
			req.After.Name,
		)
	}

	return &pb.ObjectSetResponse{
		Object: changed,
	}, nil
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

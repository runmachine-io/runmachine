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

func (s *Server) ObjectList(
	req *pb.ObjectListRequest,
	stream pb.RunmMetadata_ObjectListServer,
) error {
	if err := checkSession(req.Session); err != nil {
		return err
	}
	any := make([]*storage.PartitionObjectFilter, 0)
	for _, filter := range req.Any {
		if pfs, err := s.expandObjectFilter(req.Session, filter); err != nil {
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
		if len(req.Any) == 0 {
			// At least one filter should have been expanded
			defFilter, err := s.defaultObjectFilter(req.Session)
			if err != nil {
				return ErrFailedExpandObjectFilters
			}
			any = append(any, defFilter)
		} else {
			// The user asked for object types that don't exist, partitions
			// that don't exist, etc.
			return nil
		}
	}

	cur, err := s.store.ObjectList(any)
	if err != nil {
		return err
	}
	defer cur.Close()
	for cur.Next() {
		msg := &pb.Object{}
		if err = cur.Scan(msg); err != nil {
			return err
		}
		if err = stream.Send(msg); err != nil {
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

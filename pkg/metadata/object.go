package metadata

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/proto"
)

func (s *Server) ObjectDelete(
	ctx context.Context,
	req *pb.ObjectDeleteRequest,
) (*pb.ObjectDeleteResponse, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}
	if len(req.Any) == 0 {
		return nil, ErrAtLeastOneObjectFilterRequired
	}

	filters, err := s.normalizeObjectFilters(req.Session, req.Any)
	if err != nil {
		return nil, err
	}
	// Be extra-careful not to pass empty filters since that will delete all
	// objects...
	if len(filters) == 0 {
		return nil, ErrAtLeastOneObjectFilterRequired
	}

	cur, err := s.store.ObjectList(filters)
	if err != nil {
		return nil, err
	}
	defer cur.Close()

	resErrors := make([]string, 0)
	numDeleted := uint64(0)
	for cur.Next() {
		obj := &pb.Object{}
		if err = cur.Scan(obj); err != nil {
			return nil, err
		}
		if err = s.store.ObjectDelete(obj); err != nil {
			resErrors = append(resErrors, err.Error())
		}
		numDeleted += 1
	}
	resp := &pb.ObjectDeleteResponse{
		Errors:     resErrors,
		NumDeleted: numDeleted,
	}
	if len(resErrors) == 0 {
		return resp, ErrObjectDeleteFailed
	}
	return resp, nil
}

func (s *Server) ObjectGet(
	ctx context.Context,
	req *pb.ObjectGetRequest,
) (*pb.Object, error) {
	if err := checkSession(req.Session); err != nil {
		return nil, err
	}
	if req.Search == nil {
		return nil, ErrObjectFilterRequired
	}

	pfs, err := s.expandObjectFilter(req.Session, req.Search)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve object with search filter %s: %s",
			req.Search,
			err,
		)
		return nil, ErrUnknown
	}
	if len(pfs) == 0 {
		return nil, ErrFailedExpandObjectFilters
	}

	cur, err := s.store.ObjectList(pfs)
	if err != nil {
		return nil, err
	}
	defer cur.Close()

	found := false
	obj := &pb.Object{}
	for cur.Next() {
		if found {
			return nil, ErrMultipleRecordsFound
		}
		if err = cur.Scan(obj); err != nil {
			return nil, err
		}
		found = true
	}
	return obj, nil
}

func (s *Server) ObjectList(
	req *pb.ObjectListRequest,
	stream pb.RunmMetadata_ObjectListServer,
) error {
	if err := checkSession(req.Session); err != nil {
		return err
	}

	filters, err := s.normalizeObjectFilters(req.Session, req.Any)
	if err != nil {
		return err
	}

	cur, err := s.store.ObjectList(filters)
	if err != nil {
		return err
	}
	defer cur.Close()
	for cur.Next() {
		obj := &pb.Object{}
		if err = cur.Scan(obj); err != nil {
			return err
		}
		if err = stream.Send(obj); err != nil {
			return err
		}
	}
	return nil
}

// validateObjectSetRequest ensures that the data the user sent in the
// request's Before and After elements makes sense and meets things like
// property schema validation checks.
func (s *Server) validateObjectSetRequest(
	req *pb.ObjectSetRequest,
) error {
	after := req.After

	// Simple input data validations
	if after.ObjectType == "" {
		return ErrObjectTypeRequired
	}
	if after.Partition == "" {
		return ErrPartitionRequired
	}

	// Validate the referred to type, partition and project actually exist
	p, err := s.store.PartitionGet(after.Partition)
	if err != nil {
		if err == errors.ErrNotFound {
			return errPartitionNotFound(after.Partition)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed when validating partition in object set: %s", err)
		return errors.ErrUnknown
	}
	after.Partition = p.Uuid

	ot, err := s.store.ObjectTypeGet(after.ObjectType)
	if err != nil {
		if err == errors.ErrNotFound {
			return errObjectTypeNotFound(after.ObjectType)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed when validating object type in object set: %s", err)
		return errors.ErrUnknown
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
	return nil
}

func (s *Server) ObjectSet(
	ctx context.Context,
	req *pb.ObjectSetRequest,
) (*pb.ObjectSetResponse, error) {
	// TODO(jaypipes): AUTHZ check if user can write objects

	err := s.validateObjectSetRequest(req)
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
		changed, err = s.store.ObjectCreate(req.After)
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

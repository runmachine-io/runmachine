package metadata

import (
	"context"

	yaml "gopkg.in/yaml.v2"

	"github.com/runmachine-io/runmachine/pkg/api/types"
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
		// TODO(jaypipes): Send an event notification
		s.log.L1("user %s deleted object with UUID %s", req.Session.User, obj.Uuid)
		numDeleted += 1
	}
	resp := &pb.ObjectDeleteResponse{
		Errors:     resErrors,
		NumDeleted: numDeleted,
	}
	if len(resErrors) > 0 {
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
// request's payload can be unmarshal'd properly into YAML, contains all
// relevant fields.  and meets things like property schema validation checks.
//
// Returns a fully validated Object protobuffer message that is ready to send
// to backend storage.
func (s *Server) validateObjectSetRequest(
	req *pb.ObjectSetRequest,
) (*pb.Object, error) {

	// reads the supplied buffer which contains a YAML document describing the
	// object to create or update, and returns a pointer to an Object
	// protobuffer message containing the fields to set on the new (or changed)
	// object.
	obj := &types.Object{}
	if err := yaml.Unmarshal(req.Payload, obj); err != nil {
		return nil, err
	}

	// Simple input data validations
	if obj.Type == "" {
		return nil, ErrObjectTypeRequired
	}
	if obj.Partition == "" {
		return nil, ErrPartitionRequired
	}

	// Validate the referred to type, partition and project actually exist
	p, err := s.store.PartitionGet(obj.Partition)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errPartitionNotFound(obj.Partition)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed when validating partition in object set: %s", err)
		return nil, errors.ErrUnknown
	}
	partUuid := p.Uuid

	ot, err := s.store.ObjectTypeGet(obj.Type)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errObjectTypeNotFound(obj.Type)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed when validating object type in object set: %s", err)
		return nil, errors.ErrUnknown
	}
	objTypeCode := ot.Code

	if obj.Uuid == "" {
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

	return &pb.Object{
		// The server actually will translate partition names to UUIDs...
		Partition: partUuid,
		Type:      objTypeCode,
		Project:   obj.Project,
		Name:      obj.Name,
	}, nil
}

func (s *Server) ObjectSet(
	ctx context.Context,
	req *pb.ObjectSetRequest,
) (*pb.ObjectSetResponse, error) {
	// TODO(jaypipes): AUTHZ check if user can write objects

	obj, err := s.validateObjectSetRequest(req)
	if err != nil {
		return nil, err
	}

	var changed *pb.Object
	if obj.Uuid == "" {
		s.log.L3(
			"creating new object of type %s in partition %s with name %s...",
			obj.Type,
			obj.Partition,
			obj.Name,
		)
		changed, err = s.store.ObjectCreate(obj)
		if err != nil {
			return nil, err
		}
		s.log.L1(
			"created new object with UUID %s of type %s in partition %s with name %s",
			changed.Uuid,
			obj.Type,
			obj.Partition,
			obj.Name,
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

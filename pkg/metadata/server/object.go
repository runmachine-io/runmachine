package server

import (
	"context"

	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/conditions"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
)

func (s *Server) ObjectDelete(
	ctx context.Context,
	req *pb.ObjectDeleteRequest,
) (*pb.DeleteResponse, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}
	if len(req.Uuids) == 0 {
		return nil, ErrAtLeastOneUuidRequired
	}

	// TODO(jaypipes): Have a single filter for a list of UUIDs...
	conds := make([]*conditions.ObjectCondition, len(req.Uuids))
	for x, uuid := range req.Uuids {
		conds[x] = &conditions.ObjectCondition{
			UuidCondition: conditions.UuidEqual(uuid),
		}
	}
	owrs, err := s.store.ObjectListWithReferences(conds)
	if err != nil {
		return nil, err
	}

	numDeleted := uint64(0)
	for _, owr := range owrs {
		if err = s.store.ObjectDelete(owr); err != nil {
			return nil, err
		}
		// TODO(jaypipes): Send an event notification
		s.log.L1(
			"user %s deleted object with UUID %s",
			req.Session.User,
			owr.Object.Uuid,
		)
		numDeleted += 1
	}
	return &pb.DeleteResponse{
		NumDeleted: numDeleted,
	}, nil
}

func (s *Server) ObjectGetByUuid(
	ctx context.Context,
	req *pb.ObjectGetByUuidRequest,
) (*pb.Object, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}
	uuid := req.Uuid
	if uuid == "" {
		return nil, ErrUuidRequired
	}

	obj, err := s.store.ObjectGetByUuid(uuid)
	if err != nil {
		return nil, err
	}

	// Check that the object is in the user's Session partition and project,
	// and if not, return ErrNotFound.
	// TODO(jaypipes): Allow not checking this if the user is in a specific
	// role -- i.e. an admin?
	if obj.Partition != req.Session.Partition {
		s.log.L3(
			"found object with UUID '%s' but its partition '%s' did not "+
				"match user's Session partition '%s'",
			uuid, obj.Partition, req.Session.Partition,
		)
		return nil, ErrNotFound
	}
	// TODO(jaypipes): Make a simple cached utility for determining the scope
	// of an object type by object type code
	objType, err := s.store.ObjectTypeGetByCode(obj.ObjectType)
	if err != nil {
		return nil, err
	}
	if objType.Scope == pb.ObjectTypeScope_PROJECT &&
		obj.Project != req.Session.Project {
		s.log.L3(
			"found object with UUID '%s' but its project '%s' did not "+
				"match user's Session project '%s'",
			uuid, obj.Project, req.Session.Project,
		)
		return nil, ErrNotFound
	}

	return obj, nil
}

func (s *Server) ObjectGet(
	ctx context.Context,
	req *pb.ObjectGetRequest,
) (*pb.Object, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}
	if req.Filter == nil {
		return nil, ErrObjectFilterRequired
	}

	pfs, err := s.expandObjectFilter(req.Session, req.Filter)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		// We don't want to expose internal errors to the user, so just return
		// an unknown error after logging it.
		s.log.ERR(
			"failed to retrieve object with search filter %s: %s",
			req.Filter,
			err,
		)
		return nil, ErrUnknown
	}
	if len(pfs) == 0 {
		return nil, ErrFailedExpandObjectFilters
	}

	objects, err := s.store.ObjectList(pfs)
	if err != nil {
		return nil, err
	}
	if len(objects) > 1 {
		return nil, ErrMultipleRecordsFound
	} else if len(objects) == 0 {
		return nil, ErrNotFound
	}

	return objects[0], nil
}

func (s *Server) ObjectList(
	req *pb.ObjectListRequest,
	stream pb.RunmMetadata_ObjectListServer,
) error {
	if err := s.checkSession(req.Session); err != nil {
		return err
	}

	filters, err := s.normalizeObjectFilters(req.Session, req.Any)
	if err != nil {
		return err
	}

	objects, err := s.store.ObjectList(filters)
	if err != nil {
		return err
	}
	for _, obj := range objects {
		if err = stream.Send(obj); err != nil {
			return err
		}
	}
	return nil
}

// validateObjectCreateRequest ensures that the data the user sent is valid and
// all referenced projects, partitions, and object types are correct.
func (s *Server) validateObjectCreateRequest(
	req *pb.ObjectCreateRequest,
) (*types.ObjectWithReferences, error) {
	obj := req.Object

	// Simple input data validations
	if obj.ObjectType == "" {
		return nil, ErrObjectTypeRequired
	}
	if obj.Partition == "" {
		return nil, ErrPartitionRequired
	}

	// Validate the referred to type, partition and project actually exist
	part, err := s.store.PartitionGetByUuid(obj.Partition)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errPartitionNotFound(obj.Partition)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed when validating partition in object set: %s", err)
		return nil, errors.ErrUnknown
	}

	objType, err := s.store.ObjectTypeGetByCode(obj.ObjectType)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errObjectTypeNotFound(obj.ObjectType)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed when validating object type in object set: %s", err)
		return nil, errors.ErrUnknown
	}

	return &types.ObjectWithReferences{
		Partition:  part,
		ObjectType: objType,
		Object: &pb.Object{
			Partition:  part.Uuid,
			ObjectType: objType.Code,
			Project:    obj.Project,
			Name:       obj.Name,
			Uuid:       obj.Uuid,
			Tags:       obj.Tags,
			Properties: obj.Properties,
		},
	}, nil
}

func (s *Server) ObjectCreate(
	ctx context.Context,
	req *pb.ObjectCreateRequest,
) (*pb.ObjectCreateResponse, error) {
	if err := s.checkSession(req.Session); err != nil {
		return nil, err
	}
	// TODO(jaypipes): AUTHZ check if user can write objects

	input, err := s.validateObjectCreateRequest(req)
	if err != nil {
		return nil, err
	}
	s.log.L3(
		"creating new object of type %s in partition %s with name %s...",
		input.ObjectType.Code,
		input.Partition.Uuid,
		input.Object.Name,
	)
	changed, err := s.store.ObjectCreate(input)
	if err != nil {
		return nil, err
	}
	s.log.L1(
		"created new object with UUID %s of type %s in partition %s with name %s",
		changed.Object.Uuid,
		input.ObjectType.Code,
		input.Partition.Uuid,
		input.Object.Name,
	)

	return &pb.ObjectCreateResponse{
		Object: changed.Object,
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

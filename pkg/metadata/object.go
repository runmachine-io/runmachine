package metadata

import (
	"context"

	yaml "gopkg.in/yaml.v2"

	apitypes "github.com/runmachine-io/runmachine/pkg/api/types"
	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
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

	owrs, err := s.store.ObjectListWithReferences(filters)
	if err != nil {
		return nil, err
	}

	resErrors := make([]string, 0)
	numDeleted := uint64(0)
	for _, owr := range owrs {
		if err = s.store.ObjectDelete(owr); err != nil {
			resErrors = append(resErrors, err.Error())
		}
		// TODO(jaypipes): Send an event notification
		s.log.L1(
			"user %s deleted object with UUID %s",
			req.Session.User,
			owr.Object.Uuid,
		)
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
	if err := checkSession(req.Session); err != nil {
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

// validateObjectSetRequest ensures that the data the user sent in the
// request's payload can be unmarshal'd properly into YAML, contains all
// relevant fields.  and meets things like property meta validation checks.
//
// Returns a fully validated Object protobuffer message that is ready to send
// to backend storage.
func (s *Server) validateObjectSetRequest(
	req *pb.ObjectSetRequest,
) (*types.ObjectWithReferences, error) {
	// reads the supplied buffer which contains a YAML document describing the
	// object to create or update, and returns a pointer to an Object
	// protobuffer message containing the fields to set on the new (or changed)
	// object.
	obj := &apitypes.Object{}
	if err := yaml.Unmarshal(req.Payload, obj); err != nil {
		return nil, err
	}

	// Simple input data validations
	if obj.ObjectType == "" {
		return nil, ErrObjectTypeRequired
	}
	if obj.Partition == "" {
		return nil, ErrPartitionRequired
	}

	// Validate the referred to type, partition and project actually exist
	part, err := s.store.PartitionGet(obj.Partition)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errPartitionNotFound(obj.Partition)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed when validating partition in object set: %s", err)
		return nil, errors.ErrUnknown
	}

	objType, err := s.store.ObjectTypeGet(obj.ObjectType)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errObjectTypeNotFound(obj.ObjectType)
		}
		// We don't want to leak internal implementation errors...
		s.log.ERR("failed when validating object type in object set: %s", err)
		return nil, errors.ErrUnknown
	}

	objProperties := make([]*pb.Property, 0)
	if obj.Properties != nil {
		// Validate that the properties set against this object meet any schema
		// associated with that property key and object type
		for key, value := range obj.Properties {
			prop, err := s.validateObjectProperty(part, objType, key, value)
			if err != nil {
				return nil, err
			}
			objProperties = append(objProperties, prop)
		}
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
			Properties: objProperties,
		},
	}, nil
}

// validateObjectProperty ensures that the supplied key and value meet any
// defined property meta constraints that may have been defined for that
// object type and key. Returns a pointer to a Property protobuffer message.
func (s *Server) validateObjectProperty(
	partition *pb.Partition,
	objType *pb.ObjectType,
	key string,
	value string,
) (*pb.Property, error) {
	pds, err := s.store.PropertyDefinitionList(
		[]*types.PropertyDefinitionCondition{
			&types.PropertyDefinitionCondition{
				PartitionCondition: &types.PartitionCondition{
					Op:        types.OP_EQUAL,
					Partition: partition,
				},
				ObjectTypeCondition: &types.ObjectTypeCondition{
					Op:         types.OP_EQUAL,
					ObjectType: objType,
				},
				PropertyKeyCondition: &types.PropertyKeyCondition{
					Op:          types.OP_EQUAL,
					PropertyKey: key,
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if len(pds) > 0 {
		pd := pds[0]
		err := s.validateValueWithSchema(value, pd.Schema)
		if err != nil {
			return nil, errors.ErrFailedPropertyDefinitionValidation(key, err)
		}
	}
	return &pb.Property{
		Key:   key,
		Value: value,
	}, nil
}

// validateValueWithSchema returns an error if the supplied value passes the
// supplied property meta document, nil otherwise.
func (s *Server) validateValueWithSchema(
	value string,
	schema *pb.PropertySchema,
) error {
	return nil
}

func (s *Server) ObjectSet(
	ctx context.Context,
	req *pb.ObjectSetRequest,
) (*pb.ObjectSetResponse, error) {
	// TODO(jaypipes): AUTHZ check if user can write objects

	owr, err := s.validateObjectSetRequest(req)
	if err != nil {
		return nil, err
	}

	newObject := owr.Object.Uuid == ""
	if !newObject {
		// Check to see if an object with this UUID exists. If it does, we
		// switch to the update code path

	}

	var changed *types.ObjectWithReferences
	if newObject {
		s.log.L3(
			"creating new object of type %s in partition %s with name %s...",
			owr.ObjectType.Code,
			owr.Partition.Uuid,
			owr.Object.Name,
		)
		changed, err = s.store.ObjectCreate(owr)
		if err != nil {
			return nil, err
		}
		s.log.L1(
			"created new object with UUID %s of type %s in partition %s with name %s",
			changed.Object.Uuid,
			owr.ObjectType.Code,
			owr.Partition.Uuid,
			owr.Object.Name,
		)
	} else {
		s.log.L3("updating object with UUID %s", owr.Object.Uuid)
		// TODO(jaypipes): Implement update code path
	}

	return &pb.ObjectSetResponse{
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

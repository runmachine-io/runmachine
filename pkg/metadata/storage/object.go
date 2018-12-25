package storage

import (
	"fmt"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/golang/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
	"github.com/runmachine-io/runmachine/pkg/util"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	// $PARTITION/objects/ is a key namespace that has sub key namespaces that
	// index objects by name or project+name
	_OBJECTS_BY_TYPE_KEY = "objects/by-type/"
	_BY_NAME_KEY         = "by-name/"
	_BY_PROJECT_KEY      = "by-project/"
	// $ROOT/objects/by-uuid/ is a key namespace that stores valued keys where
	// the key is the object's UUID and the value is the serialized Object
	// protobuffer message
	_OBJECTS_BY_UUID_KEY = "objects/by-uuid/"
)

// objectByNameKey returns a string for the key to use for the object's name
// index. Depending on whether the supplied object's object type is
// project-scoped or not, the object's name index will contain the object's
// project along with the object type and name.
func (s *Store) objectByNameIndexKey(owr *types.ObjectWithReferences) (string, error) {
	switch owr.Type.Scope {
	case pb.ObjectTypeScope_PARTITION:
		// $PARTITION/objects/by-type/{type}/by-name/{name}
		return _PARTITIONS_KEY + owr.Partition.Uuid + "/" +
			_OBJECTS_BY_TYPE_KEY + owr.Type.Code + "/" +
			_BY_NAME_KEY + owr.Object.Name, nil
	case pb.ObjectTypeScope_PROJECT:
		// $PARTITION/objects/by-type/{type}/by-project/{project}/by-name/{name}
		return _PARTITIONS_KEY + owr.Partition.Uuid + "/" +
			_OBJECTS_BY_TYPE_KEY + owr.Type.Code + "/" +
			_BY_PROJECT_KEY + owr.Object.Project + "/" +
			_BY_NAME_KEY + owr.Object.Name, nil
	}
	return "", fmt.Errorf("Unknown object type scope: %s", owr.Type.Scope)
}

// ObjectDelete removes an object from storage along with any index records the
// object may have had. The supplied Object message is expected to have already
// been pulled from etcd storage and therefore contain an already-normalized
// UUID, a valid object type and partition, etc.
func (s *Store) ObjectDelete(
	owr *types.ObjectWithReferences,
) error {
	objByNameKey, err := s.objectByNameIndexKey(owr)
	if err != nil {
		return errors.ErrUnknown
	}
	objUuid := owr.Object.Uuid
	objByUuidKey := _OBJECTS_BY_UUID_KEY + objUuid

	ctx, cancel := s.requestCtx()
	defer cancel()

	// creates all the indexes and the objects/by-uuid/ entry using a
	// transaction that ensures if another thread modified anything underneath
	// us, we return an error
	then := []etcd.Op{
		// Delete the entry for the index by object name
		etcd.OpDelete(objByNameKey),
		// Delete the entry for the primary index by object UUID
		etcd.OpDelete(objByUuidKey),
	}
	// TODO(jaypipes): Should we put some If(...) clause in here that verifies
	// the object primary key and index entry existed? Not sure it's worth it,
	// really...
	resp, err := s.kv.Txn(ctx).Then(then...).Commit()

	if err != nil {
		s.log.ERR("storage.ObjectDelete: failed to create txn in etcd: %v", err)
		return errors.ErrUnknown
	} else if resp.Succeeded == false {
		s.log.ERR("storage.ObjectDelete: txn commit failed in etcd")
		return errors.ErrUnknown
	}
	return nil
}

// ObjectList returns a slice of pointers to Object protobuffer messages
// matching a set of supplied filters.
func (s *Store) ObjectList(
	any []*types.ObjectFilter,
) ([]*pb.Object, error) {
	if len(any) == 0 {
		return s.objectsGetAll()
	}
	// We iterate over our filters, evaluating each and OR'ing them together
	// into this map of object UUID to pb.Object message. This map is used to
	// group objects with the same UUID that match multiple filters.
	objs := make(map[string]*pb.Object, 0)

	for _, filter := range any {
		if filter.IsEmpty() {
			s.log.ERR("received empty types.ObjectFilter in ObjectList()")
			continue
		}
		// If the types.ObjectFilter contains a value for the Search field,
		// that means we need to look up objects by UUID or name (with an
		// optional prefix for the name). If no Search field is present, that
		// means that in order to evaluate this types.ObjectFilter we'll be
		// searching on ranges of objects by type, partition or project.
		filterObjs, err := s.objectsGetByFilter(filter)
		if err != nil {
			if err == errors.ErrNotFound {
				continue // Remember, we need to OR together the filters
			}
			return nil, err
		}
		for _, obj := range filterObjs {
			objs[obj.Uuid] = obj
		}
	}
	if len(objs) == 0 {
		return nil, nil
	}

	res := make([]*pb.Object, len(objs))
	x := 0
	for _, obj := range objs {
		res[x] = obj
		x += 1
	}
	return res, nil
}

// ObjectListWithReferences returns a slice of pointers to ObjectWithReference
// structs that have had Partition and ObjectType relations expanded inline.
func (s *Store) ObjectListWithReferences(
	any []*types.ObjectFilter,
) ([]*types.ObjectWithReferences, error) {
	objects, err := s.ObjectList(any)
	if err != nil {
		return nil, err
	}

	// NOTE(jaypipes): store.ObjectDelete() accepts a single argument of type
	// ObjectWithReferences. Here, we have two maps for Partition and
	// ObjectType messages that we fetch by partition UUID or object type code
	// while iterating over the objects to delete. These Partition and
	// ObjectType messages are included in the ObjectWithReferences structs
	// that are passed to ObjectDelete. Gotta love implementing joins in
	// Golang/memory... :(
	partitions := make(map[string]*pb.Partition, 0)
	objTypes := make(map[string]*pb.ObjectType, 0)
	res := make([]*types.ObjectWithReferences, len(objects))
	for x, obj := range objects {
		part, ok := partitions[obj.Partition]
		if !ok {
			part, err = s.PartitionGet(obj.Partition)
			if err != nil {
				msg := fmt.Sprintf(
					"failed to find partition %s while attempting to delete "+
						"object with UUID %s",
					obj.Partition,
					obj.Uuid,
				)
				s.log.ERR(msg)
				return nil, errors.ErrPartitionNotFound(obj.Partition)
			}
		}
		ot, ok := objTypes[obj.Type]
		if !ok {
			ot, err = s.ObjectTypeGet(obj.Type)
			if err != nil {
				msg := fmt.Sprintf(
					"failed to find object type %s while attempting to delete "+
						"object with UUID %s",
					obj.Type,
					obj.Uuid,
				)
				s.log.ERR(msg)
				return nil, errors.ErrObjectTypeNotFound(obj.Type)
			}
		}
		owr := &types.ObjectWithReferences{
			Partition: part,
			Type:      ot,
			Object:    obj,
		}
		res[x] = owr
	}
	return res, nil
}

func (s *Store) objectsGetByFilter(
	filter *types.ObjectFilter,
) ([]*pb.Object, error) {
	if filter.Uuid != nil {
		// If the filter specifies a Search and it looks like a UUID, then
		// all we need to do is grab the object from the primary
		// objects/by-uuid/ index and check that any other fields match the
		// object's fields. If so, just return the UUID
		obj, err := s.objectGetByUuid(filter.Uuid.Uuid)
		if err != nil {
			return nil, err
		}
		// The filter may have contained more matchers than UUID, so apply
		// those here too...
		if !filter.Matches(obj) {
			return nil, errors.ErrNotFound
		}
		return []*pb.Object{obj}, nil
	}
	if filter.Name != nil {
		// OK, we were asked to search for one or more objects having a
		// supplied name (optionally have the name as a prefix).
		//
		// If the object type has been specified, things can be searched
		// more efficiently because the object type's scope will tell us
		// whether the name index for the object is going to be be object
		// type and name or object type, project and name.
		//
		// If no object type was specified, we will need to do a full range
		// scan on all objects by the primary objects/by-uuid/ index and
		// manually check to see if the deserialized Object's name has the
		// requested name...
		if filter.ObjectType != nil && filter.Partition != nil {
			if filter.ObjectType.ObjectType.Scope == pb.ObjectTypeScope_PROJECT {
				if filter.Project != "" {
					// Just drop through if we don't have a project because
					// we won't be able to look up a project-scoped object
					// type when no project was specified, so we'll do the
					// less efficient range-scan sieve pattern to solve
					// this filter
					return s.objectsGetByProjectNameIndex(
						filter.Partition.Partition.Uuid,
						filter.ObjectType.ObjectType.Code,
						filter.Project,
						filter.Name.Name,
						filter.Name.Op != types.OP_EQUAL,
					)
				}
			} else {
				return s.objectsGetByNameIndex(
					filter.Partition.Partition.Uuid,
					filter.ObjectType.ObjectType.Code,
					filter.Name.Name,
					filter.Name.Op != types.OP_EQUAL,
				)
			}
		}
	}

	// This is called when we have no filter on object UUID/name or we have a
	// filter on name but not object type. We will get all objects and filter
	// out any objects that don't meet the supplied partition UUID, project and
	// object type code filters.
	objects, err := s.objectsGetAll()
	if err != nil {
		return nil, err
	}

	res := make([]*pb.Object, 0)
	for _, obj := range objects {
		// Use a sieve pattern, only adding the object to our results if it
		// passes all match expressions
		if !filter.Matches(obj) {
			continue
		}
		res = append(res, obj)
	}

	return res, nil
}

// objectGetByUuid returns an Object protobuffer message with the supplied
// object UUID
func (s *Store) objectGetByUuid(
	uuid string,
) (*pb.Object, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	key := _OBJECTS_BY_UUID_KEY + uuid

	resp, err := s.kv.Get(ctx, key)
	if resp.Count == 0 {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		s.log.ERR("error getting object by UUID(%s): %v", uuid, err)
		return nil, err
	}

	obj := &pb.Object{}
	if err = proto.Unmarshal(resp.Kvs[0].Value, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// objectsGetByProjectNameIndex returns Object messages that have a specified
// project and name (with optional prefix) in the supplied partition.
func (s *Store) objectsGetByProjectNameIndex(
	partUuid string,
	objTypeCode string,
	project string,
	objName string,
	usePrefix bool,
) ([]*pb.Object, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	kv := s.kvPartition(partUuid)
	key := _OBJECTS_BY_TYPE_KEY + objTypeCode + "/" +
		_BY_PROJECT_KEY + project + "/" +
		_BY_NAME_KEY + objName

	opts := []etcd.OpOption{}
	if usePrefix {
		opts = append(opts, etcd.WithPrefix())
	}

	resp, err := kv.Get(ctx, key, opts...)
	if resp.Count == 0 {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		s.log.ERR(
			"error getting objects of type %s by project and name(%s:%s): %v",
			objTypeCode,
			project,
			objName,
			err,
		)
		return nil, err
	}

	res := make([]*pb.Object, resp.Count)

	for x, entry := range resp.Kvs {
		obj, err := s.objectGetByUuid(string(entry.Value))
		if err != nil {
			return nil, err
		}
		res[x] = obj
	}

	return res, nil
}

// objectsGetByNameIndex returns Object messages that have a specified name
// (with optional prefix) in the supplied partition.
func (s *Store) objectsGetByNameIndex(
	partUuid string,
	objTypeCode string,
	objName string,
	usePrefix bool,
) ([]*pb.Object, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	kv := s.kvPartition(partUuid)
	key := _OBJECTS_BY_TYPE_KEY + objTypeCode + "/" + _BY_NAME_KEY + objName

	opts := []etcd.OpOption{}
	if usePrefix {
		opts = append(opts, etcd.WithPrefix())
	}

	resp, err := kv.Get(ctx, key, opts...)
	if resp.Count == 0 {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		s.log.ERR(
			"error getting objects of type %s by name(%s): %v",
			objTypeCode,
			objName,
			err,
		)
		return nil, err
	}

	res := make([]*pb.Object, resp.Count)

	for x, entry := range resp.Kvs {
		obj, err := s.objectGetByUuid(string(entry.Value))
		if err != nil {
			return nil, err
		}
		res[x] = obj
	}

	return res, nil
}

func (s *Store) objectsGetAll() ([]*pb.Object, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	resp, err := s.kv.Get(
		ctx,
		_OBJECTS_BY_UUID_KEY,
		etcd.WithPrefix(),
		// TODO(jaypipes): Factor the sorting/limiting/pagination out into a
		// separate utility
		etcd.WithSort(etcd.SortByKey, etcd.SortAscend),
	)

	if err != nil {
		s.log.ERR("error listing all objects: %v", err)
		return nil, err
	}

	if resp.Count == 0 {
		return []*pb.Object{}, nil
	}

	res := make([]*pb.Object, resp.Count)
	for x, kv := range resp.Kvs {
		msg := &pb.Object{}
		if err := proto.Unmarshal(kv.Value, msg); err != nil {
			return nil, err
		}
		res[x] = msg
	}

	return res, nil
}

// ObjectCreate puts the supplied object into backend storage, adding all the
// appropriate indexes. It returns the newly-created object.
func (s *Store) ObjectCreate(
	owr *types.ObjectWithReferences,
) (*types.ObjectWithReferences, error) {
	if owr.Object.Uuid == "" {
		owr.Object.Uuid = util.NewNormalizedUuid()
	} else {
		owr.Object.Uuid = util.NormalizeUuid(owr.Object.Uuid)
	}
	objUuid := owr.Object.Uuid

	objValue, err := proto.Marshal(owr.Object)
	if err != nil {
		s.log.ERR("failed to serialize object: %v", err)
		return nil, errors.ErrUnknown
	}

	objByNameKey, err := s.objectByNameIndexKey(owr)
	if err != nil {
		return nil, errors.ErrUnknown
	}
	objByUuidKey := _OBJECTS_BY_UUID_KEY + objUuid

	ctx, cancel := s.requestCtx()
	defer cancel()

	// creates all the indexes and the objects/by-uuid/ entry using a
	// transaction that ensures if another thread modified anything underneath
	// us, we return an error
	then := []etcd.Op{
		// Add the entry for the index by object name
		etcd.OpPut(objByNameKey, objUuid),
		// Add the entry for the primary index by object UUID
		etcd.OpPut(objByUuidKey, string(objValue)),
	}
	compare := []etcd.Cmp{
		// Ensure the object value and index by name don't yet exist
		etcd.Compare(etcd.Version(objByNameKey), "=", 0),
		etcd.Compare(etcd.Version(objByUuidKey), "=", 0),
	}
	resp, err := s.kv.Txn(ctx).If(compare...).Then(then...).Commit()

	if err != nil {
		s.log.ERR("object_create: failed to create txn in etcd: %v", err)
		return nil, errors.ErrUnknown
	} else if resp.Succeeded == false {
		return nil, errors.ErrDuplicate
	}
	return owr, nil
}

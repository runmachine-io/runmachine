package storage

import (
	"fmt"
	"strconv"
	"strings"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/golang/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/abstract"
	"github.com/runmachine-io/runmachine/pkg/cursor"
	"github.com/runmachine-io/runmachine/pkg/errors"
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

// A specialized filter class that has already looked up specific partition and
// object types (expanded from user-supplied partition and type filter
// strings). Users pass pb.ObjectFilter messages which contain optional
// pb.PartitionFilter and pb.ObjectTypeFilter messages. Those may be expanded
// (due to UsePrefix = true) to a set of partition UUIDs and/or object type
// codes. We then create zero or more of these ObjectListFilter structs
// that represent a specific filter on partition UUID and object type, along
// with the the object's name/UUID and UsePrefix flag.
type ObjectListFilter struct {
	Partition *pb.Partition
	Type      *pb.ObjectType
	Project   string
	Search    string
	UsePrefix bool
	// TODO(jaypipes): Add support for property and tag filters
}

func (f *ObjectListFilter) IsEmpty() bool {
	return f.Partition == nil && f.Type == nil && f.Project == "" && f.Search == ""
}

func (f *ObjectListFilter) String() string {
	attrMap := make(map[string]string, 0)
	if f.Partition != nil {
		attrMap["partition"] = f.Partition.Uuid
	}
	if f.Type != nil {
		attrMap["object_type"] = f.Type.Code
	}
	if f.Project != "" {
		attrMap["project"] = f.Project
	}
	if f.Search != "" {
		attrMap["search"] = f.Search
		attrMap["use_prefix"] = strconv.FormatBool(f.UsePrefix)
	}
	attrs := ""
	x := 0
	for k, v := range attrMap {
		if x > 0 {
			attrs += ","
		}
		attrs += k + "=" + v
	}
	return fmt.Sprintf("ObjectListFilter(%s)", attrs)
}

// objectByNameKey returns a string for the key to use for the object's name
// index. Depending on whether the supplied object's object type is
// project-scoped or not, the object's name index will contain the object's
// project along with the object type and name.
func (s *Store) objectByNameIndexKey(obj *pb.Object) (string, error) {
	objType, err := s.ObjectTypeGet(obj.Type)
	if err != nil {
		s.log.ERR(
			"storage.ObjectDelete: object type '%s' for object with "+
				"UUID '%s' was not valid: %s",
			obj.Type, obj.Uuid, err,
		)
		return "", errors.ErrUnknown
	}

	switch objType.Scope {
	case pb.ObjectTypeScope_PARTITION:
		// $PARTITION/objects/by-type/{type}/by-name/{name}
		return _PARTITIONS_KEY + obj.Partition + "/" +
			_OBJECTS_BY_TYPE_KEY + objType.Code + "/" +
			_BY_NAME_KEY + obj.Name, nil
	case pb.ObjectTypeScope_PROJECT:
		// $PARTITION/objects/by-type/{type}/by-project/{project}/by-name/{name}
		return _PARTITIONS_KEY + obj.Partition + "/" +
			_OBJECTS_BY_TYPE_KEY + objType.Code + "/" +
			_BY_PROJECT_KEY + obj.Project + "/" +
			_BY_NAME_KEY + obj.Name, nil
	}
	return "", fmt.Errorf("Unknown object type scope: %s", objType.Scope)
}

// ObjectDelete removes an object from storage along with any index records the
// object may have had. The supplied Object message is expected to have already
// been pulled from etcd storage and therefore contain an already-normalized
// UUID, a valid object type and partition, etc.
func (s *Store) ObjectDelete(
	obj *pb.Object,
) error {
	objByNameKey, err := s.objectByNameIndexKey(obj)
	if err != nil {
		return errors.ErrUnknown
	}
	objByUuidKey := _OBJECTS_BY_UUID_KEY + obj.Uuid

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

// ObjectTypeList returns a cursor over zero or more ObjectType
// protobuffer objects matching a set of supplied filters.
func (s *Store) ObjectList(
	any []*ObjectListFilter,
) (abstract.Cursor, error) {
	if len(any) == 0 {
		return s.objectsGetAll()
	}
	// We iterate over our filters, evaluating each and OR'ing them together
	// into this map of object UUID to pb.Object message. This map is used to
	// group objects with the same UUID that match multiple filters.
	objs := make(map[string]*pb.Object, 0)

	for _, filter := range any {
		if filter.IsEmpty() {
			s.log.ERR("received empty ObjectListFilter in ObjectList()")
			continue
		}
		// If the ObjectListFilter contains a value for the Search field,
		// that means we need to look up objects by UUID or name (with an
		// optional prefix for the name). If no Search field is present, that
		// means that in order to evaluate this ObjectListFilter we'll be
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
		return cursor.Empty(), nil
	}

	// Convert the map values into an array of proto.Message interfaces
	msgs := make([]proto.Message, len(objs))
	x := 0
	for _, obj := range objs {
		msgs[x] = obj
		x += 1
	}
	return cursor.NewFromSlicePBMessages(msgs[:x]), nil
}

func (s *Store) objectsGetByFilter(
	filter *ObjectListFilter,
) ([]*pb.Object, error) {
	if filter.Search != "" {
		if util.IsUuidLike(filter.Search) {
			// If the filter specifies a Search and it looks like a UUID, then
			// all we need to do is grab the object from the primary
			// objects/by-uuid/ index and check that any other fields match the
			// object's fields. If so, just return the UUID
			normUuid := util.NormalizeUuid(filter.Search)
			obj, err := s.objectGetByUuid(normUuid)
			if err != nil {
				return nil, err
			}
			if filter.Partition != nil {
				if obj.Partition != filter.Partition.Uuid {
					return nil, errors.ErrNotFound
				}
			}
			if filter.Type != nil {
				if obj.Type != filter.Type.Code {
					return nil, errors.ErrNotFound
				}
			}
			if filter.Project != "" && obj.Project != "" {
				if obj.Project != filter.Project {
					return nil, errors.ErrNotFound
				}
			}
			return []*pb.Object{obj}, nil
		} else {
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
			if filter.Type != nil && filter.Partition != nil {
				if filter.Type.Scope == pb.ObjectTypeScope_PROJECT {
					if filter.Project != "" {
						// Just drop through if we don't have a project because
						// we won't be able to look up a project-scoped object
						// type when no project was specified, so we'll do the
						// less efficient range-scan sieve pattern to solve
						// this filter
						return s.objectsGetByProjectNameIndex(
							filter.Partition.Uuid,
							filter.Type.Code,
							filter.Project,
							filter.Search,
							filter.UsePrefix,
						)
					}
				} else {
					return s.objectsGetByNameIndex(
						filter.Partition.Uuid,
						filter.Type.Code,
						filter.Search,
						filter.UsePrefix,
					)
				}
			}
		}
	}

	// This is called when we have no filter on object UUID/name or we have a
	// filter on name but not object type. We will get all objects and filter
	// out any objects that don't meet the supplied partition UUID, project and
	// object type code filters.
	cur, err := s.objectsGetAll()
	if err != nil {
		return nil, err
	}

	res := make([]*pb.Object, 0)
	for cur.Next() {
		obj := &pb.Object{}
		if err = cur.Scan(obj); err != nil {
			return nil, err
		}
		// Use a sieve pattern, only adding the object to our results if it
		// passes all match expressions
		if filter.Partition != nil {
			if obj.Partition != filter.Partition.Uuid {
				continue
			}
		}
		if filter.Type != nil {
			if obj.Type != filter.Type.Code {
				continue
			}
		}
		if filter.Project != "" && obj.Project != "" {
			if obj.Project != filter.Project {
				continue
			}
		}
		if filter.Search != "" {
			if filter.UsePrefix {
				if !strings.HasPrefix(obj.Name, filter.Search) {
					continue
				}
			} else {
				if obj.Name != filter.Search {
					continue
				}
			}
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

func (s *Store) objectsGetAll() (abstract.Cursor, error) {
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

	return cursor.NewFromEtcdGetResponse(resp), nil
}

// ObjectCreate puts the supplied object into backend storage, adding all the
// appropriate indexes. It returns the newly-created object.
func (s *Store) ObjectCreate(
	obj *pb.Object,
) (*pb.Object, error) {
	if obj.Uuid == "" {
		obj.Uuid = util.NewNormalizedUuid()
	} else {
		obj.Uuid = util.NormalizeUuid(obj.Uuid)
	}

	objValue, err := proto.Marshal(obj)
	if err != nil {
		s.log.ERR("failed to serialize object: %v", err)
		return nil, errors.ErrUnknown
	}

	objByNameKey, err := s.objectByNameIndexKey(obj)
	if err != nil {
		return nil, errors.ErrUnknown
	}
	objByUuidKey := _OBJECTS_BY_UUID_KEY + obj.Uuid

	ctx, cancel := s.requestCtx()
	defer cancel()

	// creates all the indexes and the objects/by-uuid/ entry using a
	// transaction that ensures if another thread modified anything underneath
	// us, we return an error
	then := []etcd.Op{
		// Add the entry for the index by object name
		etcd.OpPut(objByNameKey, obj.Uuid),
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
	return obj, nil
}

package storage

import (
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

// A specialized filter class that has pre-determined specific partition UUIDs
// and object type codes. Users pass pb.ObjectFilter messages which contain
// optional pb.PartitionFilter and pb.ObjectTypeFilter messages. Those may be
// expanded (due to UsePrefix = true) to a set of partition UUIDs and/or object
// type codes. We then create zero or more of these PartitionObjectFilter
// structs that represent a specific filter on partition UUID and object type,
// along with the the object's name/UUID and UsePrefix flag.
type PartitionObjectFilter struct {
	PartitionUuid  string
	Project        string
	ObjectTypeCode string
	Search         string
	UsePrefix      bool
	// TODO(jaypipes): Add support for property and tag filters
}

// ObjectTypeList returns a cursor over zero or more ObjectType
// protobuffer objects matching a set of supplied filters.
func (s *Store) ObjectList(
	any []*PartitionObjectFilter,
) (abstract.Cursor, error) {
	if len(any) == 0 {
		return s.objectsGetAll()
	}
	// We iterate over our filters, evaluating each and OR'ing them together
	// into a set of UUIDs we will look up in the primary
	// $ROOT/objects/by-uuid/ key namespace index
	uuids := make(map[string]bool, 0)

	for _, filter := range any {
		// If the filter specifies a Search and it looks like a UUID, then all
		// we need to do is add the object from the primary objects/by-uuid/
		// index and check that any other fields in this filter match. If so,
		// add the UUID to our set and we're good to go.
		if util.IsUuidLike(filter.Search) {
			normUuid := util.NormalizeUuid(filter.Search)
			if obj, err := s.objectGetByUuid(normUuid); err != nil {
				if err == errors.ErrNotFound {
					continue
				}
				return nil, err
			} else if obj != nil {
				if filter.PartitionUuid != "" {
					if obj.Partition != filter.PartitionUuid {
						continue
					}
				}
				if filter.Project != "" {
					if obj.Project != filter.Project {
						continue
					}
				}
				if filter.ObjectTypeCode != "" {
					if obj.ObjectType != filter.ObjectTypeCode {
						continue
					}
				}
				// Filter match, add it to the object UUID set
				uuids[normUuid] = true
				continue
			}
		}

		// TODO(jaypipes): OK, the user didn't specify an object UUID in their
		// filter, so we need to do repeated lookups into the various indexes
		// depending on what the user filtered by
	}
	if len(uuids) == 0 {
		return cursor.Empty(), nil
	}

	// Now we have our set of object UUIDs that we will fetch objects from the
	// primary index. I suppose we could do a single read on a range of UUID
	// keys and then ignore keys that aren't in our set of object UUIDs. Not
	// sure what would be faster... probably depend on the length of the key
	// range resulting from doing a min/max on the object UUID set.
	objs := make([]proto.Message, len(uuids))
	x := 0
	for uuid := range uuids {
		obj, err := s.objectGetByUuid(uuid)
		if err != nil {
			if err == errors.ErrNotFound {
				continue
			}
			return nil, err
		}
		objs[x] = obj
		x += 1
	}
	return cursor.NewFromSlicePBMessages(objs[:x]), nil
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

	var obj *pb.Object
	if err = proto.Unmarshal(resp.Kvs[0].Value, obj); err != nil {
		return nil, err
	}

	return obj, nil
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
	objType *pb.ObjectType,
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

	objByUuidKey := _OBJECTS_BY_UUID_KEY + obj.Uuid
	var objByNameKey string
	switch objType.Scope {
	case pb.ObjectTypeScope_PARTITION:
		// $PARTITION/objects/by-type/{type}/by-name/{name}
		objByNameKey = _PARTITIONS_KEY + obj.Partition + "/" +
			_OBJECTS_BY_TYPE_KEY + objType.Code + "/" +
			_BY_NAME_KEY + obj.Name
	case pb.ObjectTypeScope_PROJECT:
		// $PARTITION/objects/by-type/{type}/by-project/{project}/by-name/{name}
		objByNameKey = _PARTITIONS_KEY + obj.Partition + "/" +
			_OBJECTS_BY_TYPE_KEY + objType.Code + "/" +
			_BY_PROJECT_KEY + obj.Project + "/" +
			_BY_NAME_KEY + obj.Name
	}

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

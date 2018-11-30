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

func (f *PartitionObjectFilter) IsEmpty() bool {
	return f.PartitionUuid == "" && f.ObjectTypeCode == "" && f.Project == "" && f.Search == ""
}

func (f *PartitionObjectFilter) String() string {
	attrMap := make(map[string]string, 0)
	if f.PartitionUuid != "" {
		attrMap["partition"] = f.PartitionUuid
	}
	if f.ObjectTypeCode != "" {
		attrMap["object_type"] = f.ObjectTypeCode
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
	return fmt.Sprintf("PartitionObjectFilter(%s)", attrs)
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
	// into this map of object UUID to pb.Object message. This map is used to
	// group objects with the same UUID that match multiple filters.
	objs := make(map[string]*pb.Object, 0)

	for _, filter := range any {
		if filter.IsEmpty() {
			s.log.ERR("received empty PartitionObjectFilter in ObjectList()")
			continue
		}
		// If the PartitionObjectFilter contains a value for the Search field,
		// that means we need to look up objects by UUID or name (with an
		// optional prefix for the name). If no Search field is present, that
		// means that in order to evaluate this PartitionObjectFilter we'll be
		// searching on ranges of objects by type, partition or project.
		filterObjs, err := s.objectsGetBySearch(
			filter.ObjectTypeCode,
			filter.PartitionUuid,
			filter.Project,
			filter.Search,
			filter.UsePrefix,
		)
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

	// Now we have our set of object UUIDs that we will fetch objects from the
	// primary index. I suppose we could do a single read on a range of UUID
	// keys and then ignore keys that aren't in our set of object UUIDs. Not
	// sure what would be faster... probably depend on the length of the key
	// range resulting from doing a min/max on the object UUID set.
	msgs := make([]proto.Message, len(objs))
	x := 0
	for _, obj := range objs {
		msgs[x] = obj
		x += 1
	}
	return cursor.NewFromSlicePBMessages(msgs[:x]), nil
}

func (s *Store) objectsGetBySearch(
	matchObjectTypeCode string,
	matchPartitionUuid string,
	matchProject string,
	matchObjectSearch string,
	matchUsePrefix bool,
) ([]*pb.Object, error) {
	if matchObjectSearch != "" {
		if util.IsUuidLike(matchObjectSearch) {
			// If the filter specifies a Search and it looks like a UUID, then
			// all we need to do is grab the object from the primary
			// objects/by-uuid/ index and check that any other fields match the
			// object's fields. If so, just return the UUID
			normUuid := util.NormalizeUuid(matchObjectSearch)
			obj, err := s.objectGetByUuid(normUuid)
			if err != nil {
				return nil, err
			}
			if matchPartitionUuid != "" {
				if obj.Partition != matchPartitionUuid {
					return nil, errors.ErrNotFound
				}
			}
			if matchProject != "" {
				if obj.Project != matchProject {
					return nil, errors.ErrNotFound
				}
			}
			if matchObjectTypeCode != "" {
				if obj.ObjectType != matchObjectTypeCode {
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
			if matchObjectTypeCode != "" {
				// TODO(jaypipes)
				return []*pb.Object{}, nil
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
		if matchPartitionUuid != "" {
			if obj.Partition != matchPartitionUuid {
				continue
			}
		}
		if matchProject != "" {
			if obj.Project != matchProject {
				continue
			}
		}
		if matchObjectTypeCode != "" {
			if obj.ObjectType != matchObjectTypeCode {
				continue
			}
		}
		if matchObjectSearch != "" {
			if matchUsePrefix {
				if !strings.HasPrefix(obj.Name, matchObjectSearch) {
					continue
				}
			} else {
				if obj.Name != matchObjectSearch {
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

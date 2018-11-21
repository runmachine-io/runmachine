package storage

import (
	"github.com/golang/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/abstract"
	"github.com/runmachine-io/runmachine/pkg/cursor"
	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	// $PARTITION/objects/ is a key namespace that has sub key namespaces that
	// index objects by project+name and by UUID
	_OBJECTS_KEY = "objects/"
	// $PARTITION/objects/by-uuid/ is a key namespace that stores valued keys
	// where the key is the object's UUID and the value is the serialized
	// Object protobuffer message
	_OBJECTS_BY_UUID_KEY = "objects/by-uuid/"
)

type ObjectFilter struct {
	PartitionUuid  string
	Project        string
	ObjectTypeCode string
	ObjectName     string
	ObjectUuid     string
	// TODO(jaypipes): Add support for property and tag filters
}

// ObjectTypeList returns a cursor over zero or more ObjectType
// protobuffer objects matching a set of supplied filters.
func (s *Store) ObjectList(
	any []*ObjectFilter,
) (abstract.Cursor, error) {
	// We iterate over our filters, evaluating each and OR'ing them together
	// into a set of UUIDs we will look up in the primary
	// $ROOT/objects/by-uuid/ key namespace index
	objUuids := make(map[string]bool, 0)

	for _, filter := range any {
		// If the filter specifies an object UUID, then all we need to do is
		// grab the object from the primary objects/by-uuid/ index and check
		// that any other fields in this filter match. If so, add the UUID to
		// our set and we're good to go.
		if filter.ObjectUuid != "" {
			if obj, err := s.objectGetByUuid(filter.ObjectUuid); err != nil {
				if err == errors.ErrNotFound {
					continue
				}
				return nil, err
			} else if obj != nil {
				if filter.PartitionUuid != "" {
					if obj.PartitionUuid != filter.PartitionUuid {
						continue
					}
				}
				if filter.Project != "" {
					if obj.Project != filter.Project {
						continue
					}
				}
				if filter.ObjectTypeCode != "" {
					if obj.ObjectTypeCode != filter.ObjectTypeCode {
						continue
					}
				}
				if filter.ObjectName != "" {
					if obj.Name != filter.ObjectName {
						continue
					}
				}
				// Filter match, add it to the object UUID set
				objUuids[filter.ObjectUuid] = true
				continue
			}
		}

		// OK, the user didn't specify an object UUID in their filter, so we
		// need to do repeated lookups into the various indexes depending on
		// what the user filtered by

	}
	if len(objUuids) == 0 {
		return nil, nil
	}

	// Now we have our set of object UUIDs that we will fetch objects from the
	// primary index. I suppose we could do a single read on a range of UUID
	// keys and then ignore keys that aren't in our set of object UUIDs. Not
	// sure what would be faster... probably depend on the length of the key
	// range resulting from doing a min/max on the object UUID set.
	objs := make([]proto.Message, len(objUuids))
	x := 0
	for objUuid := range objUuids {
		obj, err := s.objectGetByUuid(objUuid)
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

func (s *Store) objectGetByUuid(
	objUuid string,
) (*pb.Object, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	key := _OBJECTS_BY_UUID_KEY + objUuid

	resp, err := s.kv.Get(ctx, key)
	if resp.Count == 0 {
		return nil, errors.ErrNotFound
	}
	if err != nil {
		s.log.ERR("error getting object by UUID(%s): %v", objUuid, err)
		return nil, err
	}

	var obj *pb.Object
	if err = proto.Unmarshal(resp.Kvs[0].Value, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

package storage

import (
	"strings"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/gogo/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	// The $PROPERTY_SCHEMAS key namespace is a shortcut to:
	// $ROOT/partitions/by-uuid/{partition_uuid}/property-schemas
	_PROPERTY_SCHEMAS_BY_TYPE_KEY = "property-schemas/by-type/"
)

// PropertySchemaGet returns a property schema by partition UUID, object type
// and property key.
func (s *Store) PropertySchemaGet(
	partUuid string,
	objType string,
	propSchemaKey string,
) (*pb.PropertySchema, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	kv := s.kvPartition(partUuid)
	key := _PROPERTY_SCHEMAS_BY_TYPE_KEY + objType + "/" + propSchemaKey

	gr, err := kv.Get(ctx, key, etcd.WithPrefix())
	if err != nil {
		s.log.ERR("error getting key %s: %v", key, err)
		return nil, err
	}
	if gr.Count == 0 {
		return nil, errors.ErrNotFound
	} else if gr.Count > 1 {
		return nil, errors.ErrMultipleRecords
	}

	obj := &pb.PropertySchema{}
	if err = proto.Unmarshal(gr.Kvs[0].Value, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// PropertySchemaList returns a slice of pointers to PropertySchema protobuffer
// messages matching a set of supplied filters.
func (s *Store) PropertySchemaList(
	any []*types.PropertySchemaFilter,
) ([]*pb.PropertySchema, error) {
	// Each filter is evaluated in an OR fashion, so we keep a hashmap of
	// property schema keys in order to return unique results
	objs := make(map[string]*pb.PropertySchema, 0)
	for _, filter := range any {
		filterObjs, err := s.propertySchemasGetByFilter(filter)
		if err != nil {
			return nil, err
		}
		for _, obj := range filterObjs {
			key := obj.Partition + ":" + obj.Type + ":" + obj.Key
			objs[key] = obj
		}
	}
	res := make([]*pb.PropertySchema, len(objs))
	x := 0
	for _, obj := range objs {
		res[x] = obj
		x += 1
	}
	return res, nil
}

// propertySchemasGetByFilter evaluates a single supplied PropertySchemaFilter
// that has been populated with a valid Partition, ObjectType and property key
// to filter by
func (s *Store) propertySchemasGetByFilter(
	filter *types.PropertySchemaFilter,
) ([]*pb.PropertySchema, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	kv := s.kvPartition(filter.Partition.Uuid)

	opts := []etcd.OpOption{
		// TODO(jaypipes): Factor the sorting/limiting/pagination out into a
		// separate utility
		etcd.WithSort(etcd.SortByKey, etcd.SortAscend),
	}

	// The filter may have a nil ObjectType. If that's the case, we're listing
	// property schemas of all object types and the sieve below will do our
	// filtering on any supplied property key. If we *do* have a non-nil
	// ObjectType in the filter, then we can ask etcd to do our filtering for
	// use using a more restrictive etcd.Get key string...
	var key string
	if filter.Type != nil {
		key = _PROPERTY_SCHEMAS_BY_TYPE_KEY + filter.Type.Code + "/"
		if filter.Search != "" {
			key += filter.Search
			if filter.UsePrefix {
				opts = append(opts, etcd.WithPrefix())
			}
		} else {
			opts = append(opts, etcd.WithPrefix())
		}
	} else {
		key = _PROPERTY_SCHEMAS_BY_TYPE_KEY
		opts = append(opts, etcd.WithPrefix())
	}

	resp, err := kv.Get(ctx, key, opts...)
	if err != nil {
		s.log.ERR("error listing property schemas: %v", err)
		return nil, err
	}
	if resp.Count == 0 {
		return []*pb.PropertySchema{}, nil
	}

	res := make([]*pb.PropertySchema, resp.Count)
	x := int64(0)
	for _, kv := range resp.Kvs {
		obj := &pb.PropertySchema{}
		if err = proto.Unmarshal(kv.Value, obj); err != nil {
			return nil, err
		}
		// See comment above about possibly have a nil ObjectType in the
		// filter. If this is the case, we need to evaluate the returned
		// schemas to see if they meet any supplied property key filter
		// values...
		if filter.Type == nil && filter.Search != "" {
			if filter.UsePrefix {
				if !strings.HasPrefix(obj.Key, filter.Search) {
					continue
				}
			} else {
				if obj.Key != filter.Search {
					continue
				}
			}
		}
		res[x] = obj
		x += 1
	}
	// We return res[:x] here because the above loop may have filtered out some
	// records and we don't want to return "empty slice slots"...
	return res[:x], nil
}

// PropertySchemaCreate writes the supplied PropertySchema object to the key at
// $PARTITION/property-schemas/by-type/{object_type}/{property_key}/{version}
func (s *Store) PropertySchemaCreate(
	obj *pb.PropertySchema,
) error {
	ctx, cancel := s.requestCtx()
	defer cancel()

	kv := s.kvPartition(obj.Partition)
	objType := obj.Type
	propSchemaKey := obj.Key
	key := _PROPERTY_SCHEMAS_BY_TYPE_KEY + objType + "/" + propSchemaKey

	value, err := proto.Marshal(obj)
	if err != nil {
		return err
	}

	// create the property schema using a transaction that ensures another
	// thread hasn't created a property schema with the same key underneath us
	onSuccess := etcd.OpPut(key, string(value))
	// Ensure the key doesn't yet exist
	compare := etcd.Compare(etcd.Version(key), "=", 0)
	resp, err := kv.Txn(ctx).If(compare).Then(onSuccess).Commit()

	if err != nil {
		s.log.ERR("failed to create txn in etcd: %v", err)
		return err
	} else if resp.Succeeded == false {
		s.log.L3("another thread already created key %s.", key)
		return errors.ErrGenerationConflict
	}
	return nil
}

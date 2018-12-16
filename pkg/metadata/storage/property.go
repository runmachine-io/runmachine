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
	// The $PROPERTY_DEFINITIONS key namespace is a shortcut to:
	// $ROOT/partitions/by-uuid/{partition_uuid}/property-definitions
	_PROPERTY_DEFINITIONS_BY_TYPE_KEY = "property-definitions/by-type/"
)

// PropertyDefinitionGet returns a property definition by partition UUID, object type
// and property key.
func (s *Store) PropertyDefinitionGet(
	partUuid string,
	objType string,
	propDefKey string,
) (*pb.PropertyDefinition, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	kv := s.kvPartition(partUuid)
	key := _PROPERTY_DEFINITIONS_BY_TYPE_KEY + objType + "/" + propDefKey

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

	obj := &pb.PropertyDefinition{}
	if err = proto.Unmarshal(gr.Kvs[0].Value, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// PropertyDefinitionList returns a slice of pointers to PropertyDefinition protobuffer
// messages matching a set of supplied filters.
func (s *Store) PropertyDefinitionList(
	any []*types.PropertyDefinitionFilter,
) ([]*pb.PropertyDefinition, error) {
	// Each filter is evaluated in an OR fashion, so we keep a hashmap of
	// property definition keys in order to return unique results
	objs := make(map[string]*pb.PropertyDefinition, 0)
	for _, filter := range any {
		filterObjs, err := s.propertyDefinitionsGetByFilter(filter)
		if err != nil {
			return nil, err
		}
		for _, obj := range filterObjs {
			key := obj.Partition + ":" + obj.Type + ":" + obj.Key
			objs[key] = obj
		}
	}
	res := make([]*pb.PropertyDefinition, len(objs))
	x := 0
	for _, obj := range objs {
		res[x] = obj
		x += 1
	}
	return res, nil
}

// propertyDefinitionsGetByFilter evaluates a single supplied PropertyDefinitionFilter
// that has been populated with a valid Partition, ObjectType and property key
// to filter by
func (s *Store) propertyDefinitionsGetByFilter(
	filter *types.PropertyDefinitionFilter,
) ([]*pb.PropertyDefinition, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	kv := s.kvPartition(filter.Partition.Uuid)

	opts := []etcd.OpOption{
		// TODO(jaypipes): Factor the sorting/limiting/pagination out into a
		// separate utility
		etcd.WithSort(etcd.SortByKey, etcd.SortAscend),
	}

	// The filter may have a nil ObjectType. If that's the case, we're listing
	// property definitions of all object types and the sieve below will do our
	// filtering on any supplied property key. If we *do* have a non-nil
	// ObjectType in the filter, then we can ask etcd to do our filtering for
	// use using a more restrictive etcd.Get key string...
	var key string
	if filter.Type != nil {
		key = _PROPERTY_DEFINITIONS_BY_TYPE_KEY + filter.Type.Code + "/"
		if filter.Search != "" {
			key += filter.Search
			if filter.UsePrefix {
				opts = append(opts, etcd.WithPrefix())
			}
		} else {
			opts = append(opts, etcd.WithPrefix())
		}
	} else {
		key = _PROPERTY_DEFINITIONS_BY_TYPE_KEY
		opts = append(opts, etcd.WithPrefix())
	}

	resp, err := kv.Get(ctx, key, opts...)
	if err != nil {
		s.log.ERR("error listing property definitions: %v", err)
		return nil, err
	}
	if resp.Count == 0 {
		return []*pb.PropertyDefinition{}, nil
	}

	res := make([]*pb.PropertyDefinition, resp.Count)
	x := int64(0)
	for _, kv := range resp.Kvs {
		obj := &pb.PropertyDefinition{}
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

// PropertyDefinitionCreate writes the supplied PropertyDefinition object to the key at
// $PARTITION/property-definitions/by-type/{object_type}/{property_key}/{version}
func (s *Store) PropertyDefinitionCreate(
	obj *types.PropertyDefinitionWithReferences,
) error {
	ctx, cancel := s.requestCtx()
	defer cancel()

	partUuid := obj.Partition.Uuid
	objType := obj.Type.Code
	propDefKey := obj.Definition.Key
	kv := s.kvPartition(partUuid)
	key := _PROPERTY_DEFINITIONS_BY_TYPE_KEY + objType + "/" + propDefKey

	value, err := proto.Marshal(obj.Definition)
	if err != nil {
		return err
	}

	// create the property definition using a transaction that ensures another
	// thread hasn't created a property definition with the same key underneath us
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

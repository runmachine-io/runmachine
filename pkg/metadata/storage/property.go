package storage

import (
	"fmt"

	etcd "github.com/coreos/etcd/clientv3"
	etcd_namespace "github.com/coreos/etcd/clientv3/namespace"
	"github.com/gogo/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/abstract"
	"github.com/runmachine-io/runmachine/pkg/cursor"
	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	_PROPERTY_SCHEMAS_KEY = "property-schemas/"
)

func (s *Store) kvPropertySchemas(
	partition string,
) etcd.KV {
	// The $PROPERTY_SCHEMAS key namespace is a shortcut to:
	// $ROOT/partitions/by-uuid/{partition_uuid}/property-schemas
	return etcd_namespace.NewKV(
		s.kvPartition(partition),
		_PROPERTY_SCHEMAS_KEY,
	)
}

// PropertySchemaGet returns a property schema by partition UUID, object type
// and property key.
func (s *Store) PropertySchemaGet(
	partUuid string,
	objType string,
	propSchemaKey string,
) (*pb.PropertySchema, error) {
	kv := s.kvPropertySchemas(partUuid)
	ctx, cancel := s.requestCtx()
	defer cancel()
	key := fmt.Sprintf("by-type/%s/%s", objType, propSchemaKey)
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
	var obj *pb.PropertySchema
	if err = proto.Unmarshal(gr.Kvs[0].Value, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// PropertySchemaList returns a cursor over zero or more PropertySchema
// protobuffer objects matching a set of supplied filters.
func (s *Store) PropertySchemaList(
	any []*types.PropertySchemaFilter,
) (abstract.Cursor, error) {
	// Iterate over the partitions in our filter list
	partUuids := make([]string, 0)
	for _, filter := range any {
		if filter.Partition != nil {
			partUuids = append(partUuids, filter.Partition.Uuid)
		}
	}
	if len(partUuids) == 0 {
		// TODO(jaypipes): Grab the list of all known partition UUIDs. Note
		// that the metadata server will have already verified the calling user
		// has the ability to see all partitions.
	}
	for _, partUuid := range partUuids {
		// TODO(jaypipes): Merge all returned getters into a single cursor
		return s.propertySchemaGetFilteredByPartition(partUuid)
	}
	return cursor.Empty(), nil
}

func (s *Store) propertySchemaGetFilteredByPartition(
	partUuid string,
) (abstract.Cursor, error) {
	kv := s.kvPropertySchemas(partUuid)
	ctx, cancel := s.requestCtx()
	defer cancel()
	// TODO(jaypipes): Handle the any filters
	resp, err := kv.Get(
		ctx,
		"/",
		etcd.WithPrefix(),
		// TODO(jaypipes): Factor the sorting/limiting/pagination out into a
		// separate utility
		etcd.WithSort(etcd.SortByKey, etcd.SortAscend),
	)
	if err != nil {
		s.log.ERR("error listing property schemas: %v", err)
		return nil, err
	}

	return cursor.NewFromEtcdGetResponse(resp), nil
}

// PropertySchemaCreate writes the supplied PropertySchema object to the key at
// $PARTITION/property-schemas/by-type/{object_type}/{property_key}/{version}
func (s *Store) PropertySchemaCreate(
	obj *pb.PropertySchema,
) error {
	kv := s.kvPropertySchemas(obj.Partition)
	ctx, cancel := s.requestCtx()
	defer cancel()

	objType := obj.Type
	propSchemaKey := obj.Key

	key := fmt.Sprintf("by-type/%s/%s", objType, propSchemaKey)
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

package storage

import (
	"fmt"

	etcd "github.com/coreos/etcd/clientv3"
	etcd_namespace "github.com/coreos/etcd/clientv3/namespace"
	"github.com/gogo/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/abstract"
	"github.com/runmachine-io/runmachine/pkg/cursor"
	"github.com/runmachine-io/runmachine/pkg/errors"
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

func (s *Store) PropertySchemaGet(
	partition string,
	objType string,
	propSchemaKey string,
	version uint32,
) (*pb.PropertySchema, error) {
	kv := s.kvPropertySchemas(partition)
	ctx, cancel := s.requestCtx()
	defer cancel()
	key := fmt.Sprintf("by-type/%s/%s/%d", objType, propSchemaKey, version)
	gr, err := kv.Get(ctx, key, etcd.WithPrefix())
	if err != nil {
		s.log.ERR("error getting key %s: %v", key, err)
		return nil, err
	}
	nKeys := len(gr.Kvs)
	if nKeys == 0 {
		return nil, errors.ErrNotFound
	} else if nKeys > 1 {
		return nil, errors.ErrMultipleRecords
	}
	var obj *pb.PropertySchema
	if err = proto.Unmarshal(gr.Kvs[0].Value, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func (s *Store) PropertySchemaList(
	req *pb.PropertySchemaListRequest,
) (abstract.Cursor, error) {
	partition := req.Session.Partition.Uuid
	if req.Filters != nil {
		if len(req.Filters.Partitions) > 0 {
			// TODO(jaypipes): loop through all searched-for partitions
			partition = req.Filters.Partitions[0].Uuid
		}
	}
	kv := s.kvPropertySchemas(partition)
	ctx, cancel := s.requestCtx()
	defer cancel()
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

	return cursor.NewEtcdPBCursor(resp), nil
}

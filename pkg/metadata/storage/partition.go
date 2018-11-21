package storage

import (
	etcd "github.com/coreos/etcd/clientv3"
	etcd_namespace "github.com/coreos/etcd/clientv3/namespace"
	"github.com/gogo/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/abstract"
	"github.com/runmachine-io/runmachine/pkg/cursor"
	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	// The key namespace with partition-specific information. Has two sub key
	// namespaces, one is an index on the partition's name with valued keys
	// pointing to UUIDs of that partition. The other is a set of valued keys
	// listed by UUID with values containing Partition protobuffer records.
	_PARTITIONS_KEY = "partitions/"
	// The index into partition UUIDs by name
	_PARTITIONS_BY_NAME_KEY = "partitions/by-name/"
	// The index into Partition protobuffer objects by UUID
	_PARTITIONS_BY_UUID_KEY = "partitions/by-uuid/"
)

// kvPartition returns an etcd.KV that is nahespaced to a specific partition.
// We use the nomenclature $PARTITION to refer to this key namespace.
// $PARTITION refers to $ROOT/partitions/by-uuid/{partition_uuid}/
func (s *Store) kvPartition(partUuid string) etcd.KV {
	key := _PARTITIONS_BY_UUID_KEY + partUuid + "/"
	return etcd_namespace.NewKV(s.kv, key)
}

// PartitionGet returns a Partition protobuffer message that has the UUID or
// name of the supplied search string
func (s *Store) PartitionGet(
	search string,
) (*pb.Partition, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	// TODO(jaypipes): This is going to be a common pattern (look up by UUID,
	// fall back to looking up by name index and following the index pointer to
	// the UUID data record. Look for a way to DRY this up

	// First try looking up the partition by UUID. If not, match, then we try
	// the by-name index...
	byUuidKey := _PARTITIONS_BY_UUID_KEY + search
	var resp *etcd.GetResponse
	resp, err := s.kv.Get(ctx, byUuidKey)
	if err != nil {
		s.log.ERR("error getting key %s: %v", byUuidKey, err)
		return nil, err
	}

	if resp.Count == 0 {
		byNameKey := _PARTITIONS_BY_NAME_KEY + search
		resp, err = s.kv.Get(ctx, byNameKey)
		if err != nil {
			s.log.ERR("error getting key %s: %v", byNameKey, err)
			return nil, err
		}
		if resp.Count == 0 {
			return nil, errors.ErrNotFound
		}
		partUuid := resp.Kvs[0].Value
		byUuidKey = _PARTITIONS_BY_UUID_KEY + string(partUuid)
		resp, err = s.kv.Get(ctx, byUuidKey)
		if err != nil {
			s.log.ERR("error getting key %s: %v", byUuidKey, err)
			return nil, err
		}

		if resp.Count == 0 {
			// NOTE(jaypipes): This is a major data corruption, since we have
			// an index record by the partition name pointing to this UUID but
			// no data record for the UUID...
			s.log.ERR("DATA CORRUPTION! %s exists but no data record at %s", byNameKey, byUuidKey)
			return nil, errors.ErrNotFound
		}
	}
	obj := &pb.Partition{}
	if err = proto.Unmarshal(resp.Kvs[0].Value, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// PartitionList returns a cursor that may be used to iterate over Partition
// protobuffer objects stored in etcd
func (s *Store) PartitionList(
	req *pb.PartitionListRequest,
) (abstract.Cursor, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()
	resp, err := s.kv.Get(
		ctx,
		_PARTITIONS_BY_UUID_KEY,
		etcd.WithPrefix(),
		// Since each partition will have a key namespace equal to
		// (_PARTITIONS_BY_UUID_KEY + {UUID} + "/") we need to limit our search
		// range to end with only the fixed 32 characters that all UUID values
		// comprise. This allows $ROOT/partitions/by-uuid/{UUID} to be returned
		// but prevents $ROOT/partitions/by-uuid/{UUID}/ from being returned.
		// The latter is the partition key namespace. The former is the
		// partition key that contains a serialized Protobuffer object.
		etcd.WithRange(_PARTITIONS_BY_UUID_KEY+_MAX_UUID),
		// TODO(jaypipes): Factor the sorting/limiting/pagination out into a
		// separate utility
		etcd.WithSort(etcd.SortByKey, etcd.SortAscend),
	)

	if err != nil {
		s.log.ERR("error listing partitions: %v", err)
		return nil, err
	}

	return cursor.NewFromEtcdGetResponse(resp), nil
}

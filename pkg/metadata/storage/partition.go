package storage

import (
	"fmt"

	etcd "github.com/coreos/etcd/clientv3"
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
	_PARTITIONS_BY_NAME_KEY = "partitions/by-name/%s"
	// The index into Partition protobuffer objects by UUID
	// $PARTITION refers to the key namespace at
	// $ROOT/partitions/by-uuid/{partition_uuid}
	_PARTITIONS_BY_UUID_KEY = "partitions/by-uuid/%s"
)

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
	byUuidKey := fmt.Sprintf(_PARTITIONS_BY_UUID_KEY, search)
	grByUuid, err := s.kv.Get(ctx, byUuidKey)
	if err != nil {
		s.log.ERR("error getting key %s: %v", byUuidKey, err)
		return nil, err
	}

	if grByUuid.Count == 0 {
		byNameKey := fmt.Sprintf(_PARTITIONS_BY_NAME_KEY, search)
		grByName, err := s.kv.Get(ctx, byNameKey)
		if err != nil {
			s.log.ERR("error getting key %s: %v", byNameKey, err)
			return nil, err
		}
		if grByName.Count == 0 {
			return nil, errors.ErrNotFound
		}
		partUuid := grByName.Kvs[0].Value
		byUuidKey = fmt.Sprintf(_PARTITIONS_BY_UUID_KEY, partUuid)
		grByUuid, err = s.kv.Get(ctx, byUuidKey)
		if err != nil {
			s.log.ERR("error getting key %s: %v", byUuidKey, err)
			return nil, err
		}

		if grByUuid.Count == 0 {
			// NOTE(jaypipes): This is a major data corruption, since we have
			// an index record by the partition name pointing to this UUID but
			// no data record for the UUID...
			s.log.ERR("DATA CORRUPTION! %s exists but no data record at %s", byNameKey, byUuidKey)
		}
	}
	obj := &pb.Partition{}
	if err = proto.Unmarshal(grByUuid.Kvs[0].Value, obj); err != nil {
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
		_PARTITIONS_KEY,
		etcd.WithPrefix(),
		// TODO(jaypipes): Factor the sorting/limiting/pagination out into a
		// separate utility
		etcd.WithSort(etcd.SortByKey, etcd.SortAscend),
	)
	if err != nil {
		s.log.ERR("error listing partitions: %v", err)
		return nil, err
	}

	return cursor.NewEtcdPBCursor(resp), nil
}

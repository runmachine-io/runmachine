package storage

import (
	etcd "github.com/coreos/etcd/clientv3"
	etcd_namespace "github.com/coreos/etcd/clientv3/namespace"
	"github.com/golang/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/abstract"
	"github.com/runmachine-io/runmachine/pkg/cursor"
	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/util"
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

// partitionGetByUuid returns a Partition protobuffer object with the supplied UUID
func (s *Store) partitionGetByUuid(
	uuid string,
) (*pb.Partition, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()
	key := _PARTITIONS_BY_UUID_KEY + util.NormalizeUuid(uuid)
	resp, err := s.kv.Get(ctx, key)
	if err != nil {
		s.log.ERR("error getting partition by UUID(%s): %v", key, err)
		return nil, err
	}
	if resp.Count == 0 {
		return nil, errors.ErrNotFound
	}
	obj := &pb.Partition{}
	if err = proto.Unmarshal(resp.Kvs[0].Value, obj); err != nil {
		return nil, err
	}
	return obj, nil
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
	if util.IsUuidLike(search) {
		p, err := s.partitionGetByUuid(search)
		if p != nil {
			return p, nil
		}
		if err != nil && err != errors.ErrNotFound {
			return nil, err
		}
	}

	byNameKey := _PARTITIONS_BY_NAME_KEY + search
	resp, err := s.kv.Get(ctx, byNameKey)
	if err != nil {
		s.log.ERR("error getting key %s: %v", byNameKey, err)
		return nil, err
	}
	if resp.Count == 0 {
		return nil, errors.ErrNotFound
	}
	partUuid := string(resp.Kvs[0].Value)
	p, err := s.partitionGetByUuid(partUuid)
	if p != nil {
		return p, nil
	}
	if err != nil {
		if err == errors.ErrNotFound {
			// NOTE(jaypipes): This is a major data corruption, since we have
			// an index record by the partition name pointing to this UUID but
			// no data record for the UUID...
			s.log.ERR("DATA CORRUPTION! %s exists but no data record at partitions/by-uuid/%s", byNameKey, partUuid)
		}
		return nil, err
	}
	return nil, nil
}

// PartitionList returns a cursor that may be used to iterate over Partition
// protobuffer objects stored in etcd
func (s *Store) PartitionList(
	any []*pb.PartitionFilter,
) (abstract.Cursor, error) {
	if len(any) == 0 {
		return s.partitionGetAll()
	}

	// OK, we've got some filters so we need to process each filter, OR'ing
	// them together to form a result. For each filter, we evaluate whether the
	// user has specified a UUID for the search term, in which case we just
	// grab the partition by UUID. If not, we look up partitions by name,
	// optionally using a prefix if the supplied filter indicates to use
	// prefixing.
	uuids := make(map[string]bool, 0)

	for _, filter := range any {
		// If the filter specifies a UUID search term, then just add it to our
		// list of partitions to grab by UUID. If not, look up any partitions
		// having the supplied name, with optional prefix.
		if util.IsUuidLike(filter.Search) {
			uuids[filter.Search] = true
			continue
		}

		uuidsByName, err := s.partitionUuidsGetByName(
			filter.Search,
			filter.UsePrefix,
		)
		if err != nil {
			if err == errors.ErrNotFound {
				continue
			}
			return nil, err
		}
		for _, uuid := range uuidsByName {
			uuids[uuid] = true
		}

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
		obj, err := s.partitionGetByUuid(uuid)
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

// partitionUuidsGetByName returns a slice of strings with all partition UUIDs
// have a supplied name
func (s *Store) partitionUuidsGetByName(
	search string,
	usePrefix bool,
) ([]string, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	key := _PARTITIONS_BY_NAME_KEY + search

	opts := []etcd.OpOption{
		// TODO(jaypipes): Factor the sorting/limiting/pagination out into a
		// separate utility
		etcd.WithSort(etcd.SortByKey, etcd.SortAscend),
	}

	if usePrefix {
		opts = append(opts, etcd.WithPrefix())
	}

	resp, err := s.kv.Get(ctx, key, opts...)
	if err != nil {
		s.log.ERR("error listing partitions by name: %v", err)
		return nil, err
	}

	if resp.Count == 0 {
		return nil, errors.ErrNotFound
	}

	res := make([]string, resp.Count)
	for x := int64(0); x < resp.Count; x++ {
		res[x] = string(resp.Kvs[x].Value)
	}
	return res, nil
}

func (s *Store) partitionGetAll() (abstract.Cursor, error) {
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
		s.log.ERR("error listing all partitions: %v", err)
		return nil, err
	}

	return cursor.NewFromEtcdGetResponse(resp), nil
}

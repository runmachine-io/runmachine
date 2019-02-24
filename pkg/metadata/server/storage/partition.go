package storage

import (
	etcd "github.com/coreos/etcd/clientv3"
	etcd_namespace "github.com/coreos/etcd/clientv3/namespace"
	"github.com/golang/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/runmachine-io/runmachine/pkg/util"
)

const (
	// The key namespace with partition-specific information. Has two sub key
	// namespaces, one is an index on the partition's name with valued keys
	// pointing to UUIDs of that partition. The other is a set of valued keys
	// listed by UUID with values containing Partition protobuffer records.
	// In addition to the indexes, each partition's objects are contained in
	// the $ROOT/partitions/{uuid}/ key namespace.
	_PARTITIONS_KEY = "partitions/"
	// The index into partition UUIDs by name
	_PARTITIONS_BY_NAME_KEY = "partitions/by-name/"
	// The index into Partition protobuffer objects by UUID
	_PARTITIONS_BY_UUID_KEY = "partitions/by-uuid/"
)

// kvPartition returns an etcd.KV that is namespaced to a specific partition.
// We use the nomenclature $PARTITION to refer to this key namespace.
// $PARTITION refers to $ROOT/partitions/{partition_uuid}/
func (s *Store) kvPartition(partUuid string) etcd.KV {
	key := _PARTITIONS_KEY + partUuid + "/"
	return etcd_namespace.NewKV(s.kv, key)
}

// PartitionGetByUuid returns a Partition protobuffer message with the supplied
// UUID
func (s *Store) PartitionGetByUuid(
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

// PartitionGetByName returns a Partition protobuffer message with the supplied
// name
func (s *Store) PartitionGetByName(
	name string,
) (*pb.Partition, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()
	byNameKey := _PARTITIONS_BY_NAME_KEY + name
	resp, err := s.kv.Get(ctx, byNameKey)
	if err != nil {
		s.log.ERR("error getting key %s: %v", byNameKey, err)
		return nil, err
	}
	if resp.Count == 0 {
		return nil, errors.ErrNotFound
	}
	partUuid := string(resp.Kvs[0].Value)
	p, err := s.PartitionGetByUuid(partUuid)
	if p != nil {
		return p, nil
	}
	if err != nil {
		if err == errors.ErrNotFound {
			// NOTE(jaypipes): This is a major data corruption, since we have
			// an index record by the partition name pointing to this UUID but
			// no data record for the UUID...
			s.log.ERR(
				"DATA CORRUPTION! %s exists but no data record at "+
					"partitions/by-uuid/%s",
				byNameKey,
				partUuid,
			)
		}
		return nil, err
	}
	return nil, nil
}

// PartitionList returns a cursor that may be used to iterate over Partition
// protobuffer objects stored in etcd
func (s *Store) PartitionList(
	any []*pb.PartitionFilter,
) ([]*pb.Partition, error) {
	if len(any) == 0 {
		return s.partitionsGetAll()
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
		if filter.UuidFilter != nil {
			uuids[filter.UuidFilter.Uuid] = true
			continue
		}

		uuidsByName, err := s.partitionUuidsGetByName(
			filter.NameFilter.Name,
			filter.NameFilter.UsePrefix,
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
		return []*pb.Partition{}, nil
	}

	// Now we have our set of object UUIDs that we will fetch objects from the
	// primary index. I suppose we could do a single read on a range of UUID
	// keys and then ignore keys that aren't in our set of object UUIDs. Not
	// sure what would be faster... probably depend on the length of the key
	// range resulting from doing a min/max on the object UUID set.
	res := make([]*pb.Partition, len(uuids))
	x := 0
	for uuid := range uuids {
		obj, err := s.PartitionGetByUuid(uuid)
		if err != nil {
			if err == errors.ErrNotFound {
				continue
			}
			return nil, err
		}
		res[x] = obj
		x += 1
	}
	return res[:x], nil
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
	for x, kv := range resp.Kvs {
		res[x] = string(kv.Value)
	}
	return res, nil
}

func (s *Store) partitionsGetAll() ([]*pb.Partition, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	resp, err := s.kv.Get(
		ctx,
		_PARTITIONS_BY_UUID_KEY,
		etcd.WithPrefix(),
		// TODO(jaypipes): Factor the sorting/limiting/pagination out into a
		// separate utility
		etcd.WithSort(etcd.SortByKey, etcd.SortAscend),
	)

	if err != nil {
		s.log.ERR("error listing partitions: %v", err)
		return nil, err
	}
	if resp.Count == 0 {
		return []*pb.Partition{}, nil
	}

	res := make([]*pb.Partition, resp.Count)
	for x, kv := range resp.Kvs {
		msg := &pb.Partition{}
		if err := proto.Unmarshal(kv.Value, msg); err != nil {
			return nil, err
		}
		res[x] = msg
	}
	return res, nil
}

// PartitionCreate stores a new partition record in backend storage. It returns
// ErrDuplicate if a partition with the same UUID or name already exists.
// Returns the Partition that was written to storage, which may have had a UUID
// created for it.
func (s *Store) PartitionCreate(
	part *pb.Partition,
) (*pb.Partition, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	if part.Uuid == "" {
		part.Uuid = util.NewNormalizedUuid()
	} else {
		part.Uuid = util.NormalizeUuid(part.Uuid)
	}

	partByNameKey := _PARTITIONS_BY_NAME_KEY + part.Name
	partByUuidKey := _PARTITIONS_BY_UUID_KEY + part.Uuid

	partValue, err := proto.Marshal(part)
	if err != nil {
		s.log.ERR("failed to serialize partition: %v", err)
		return nil, err
	}

	// creates the partition keys using a transaction that ensures if another
	// thread modified anything underneath us, we return an error
	then := []etcd.Op{
		// Add the entry for the index by partition name
		etcd.OpPut(partByNameKey, part.Uuid),
		// Add the entry for the index by partition UUID
		etcd.OpPut(partByUuidKey, string(partValue)),
	}
	compare := []etcd.Cmp{
		// Ensure the partition value and index by name don't yet exist
		etcd.Compare(etcd.Version(partByNameKey), "=", 0),
		etcd.Compare(etcd.Version(partByUuidKey), "=", 0),
	}
	resp, err := s.kv.Txn(ctx).If(compare...).Then(then...).Commit()

	if err != nil {
		s.log.ERR("failed to create txn in etcd: %v", err)
		return nil, err
	} else if resp.Succeeded == false {
		return nil, errors.ErrDuplicate
	}
	return part, nil
}

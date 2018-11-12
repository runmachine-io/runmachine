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
	_PARTITIONS_KEY = "partitions/"
)

func (s *Store) kvPartitions() etcd.KV {
	// The $PARTITIONS key namespace is a shortcut to: $ROOT/partitions
	return etcd_namespace.NewKV(s.kv, _PARTITIONS_KEY)
}

// PartitionGet returns a Partition protobuffer message that has the UUID or
// name of the supplied search string
func (s *Store) PartitionGet(
	search string,
) (*pb.Partition, error) {
	kv := s.kvPartitions()
	ctx, cancel := s.requestCtx()
	defer cancel()

	// First try looking up the partition by UUID. If not, match, then we try
	// the by-name index...
	key := fmt.Sprintf("by-uuid/%s", search)
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
	var obj *pb.Partition
	if err = proto.Unmarshal(gr.Kvs[0].Value, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// PartitionList returns a cursor that may be used to iterate over Partition
// protobuffer objects stored in etcd
func (s *Store) PartitionList(
	req *pb.PartitionListRequest,
) (abstract.Cursor, error) {
	kv := s.kvPartitions()
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
		s.log.ERR("error listing partitions: %v", err)
		return nil, err
	}

	return cursor.NewEtcdPBCursor(resp), nil
}

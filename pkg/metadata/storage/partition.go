package storage

import (
	etcd "github.com/coreos/etcd/clientv3"
	etcd_namespace "github.com/coreos/etcd/clientv3/namespace"

	"github.com/runmachine-io/runmachine/pkg/abstract"
	"github.com/runmachine-io/runmachine/pkg/cursor"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	_PARTITIONS_KEY = "partitions/"
)

func (s *Store) kvPartitions() etcd.KV {
	// The $PARTITIONS key namespace is a shortcut to: $ROOT/partitions
	return etcd_namespace.NewKV(s.kv, _PARTITIONS_KEY)
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

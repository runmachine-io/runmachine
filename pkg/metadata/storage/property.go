package storage

import (
	etcd "github.com/coreos/etcd/clientv3"
	etcd_namespace "github.com/coreos/etcd/clientv3/namespace"

	"github.com/runmachine-io/runmachine/pkg/abstract"
	"github.com/runmachine-io/runmachine/pkg/cursor"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	_KEY_PROPERTY_SCHEMA = "property_schemas/"
)

func (s *Store) kvPropertySchemas() etcd.KV {
	return etcd_namespace.NewKV(s.kv, _KEY_PROPERTY_SCHEMA)
}

func (s *Store) PropertySchemaList(
	req *pb.PropertySchemaListRequest,
) (abstract.Cursor, error) {
	kv := s.kvPropertySchemas()
	ctx, cancel := s.requestCtx()
	resp, err := kv.Get(
		ctx,
		"/",
		etcd.WithPrefix(),
		// TODO(jaypipes): Factor the sorting/limiting/pagination out into a
		// separate utility
		etcd.WithSort(etcd.SortByKey, etcd.SortAscend),
	)
	cancel()
	if err != nil {
		s.log.ERR("error listing property schemas: %v", err)
		return nil, err
	}

	return cursor.NewEtcdPBCursor(resp), nil
}

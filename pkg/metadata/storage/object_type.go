package storage

import (
	etcd "github.com/coreos/etcd/clientv3"

	"github.com/runmachine-io/runmachine/pkg/abstract"
	"github.com/runmachine-io/runmachine/pkg/cursor"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	_OBJECT_TYPES_KEY = "object-types/"
)

// ObjectTypeList returns a cursor over zero or more ObjectType
// protobuffer objects matching a set of supplied filters.
func (s *Store) ObjectTypeList(
	any []*pb.ObjectTypeFilter,
) (abstract.Cursor, error) {
	if len(any) == 0 {
		// Just return all object types
		return s.objectTypesGetByCode("", true)
	}
	for _, filter := range any {
		// TODO(jaypipes): Merge all returned getters into a single cursor
		return s.objectTypesGetByCode(filter.Code, filter.UsePrefix)
	}
	return nil, nil
}

func (s *Store) objectTypesGetByCode(
	code string,
	usePrefix bool,
) (abstract.Cursor, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	key := _OBJECT_TYPES_KEY + code

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
		s.log.ERR("error listing object types: %v", err)
		return nil, err
	}

	return cursor.NewEtcdPBCursor(resp), nil
}

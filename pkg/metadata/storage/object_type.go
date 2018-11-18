package storage

import (
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/gogo/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/abstract"
	"github.com/runmachine-io/runmachine/pkg/cursor"
	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	_OBJECT_TYPES_KEY = "object-types/"
)

var (
	// The collection of well-known runm object types
	runmObjectTypes = []*pb.ObjectType{
		&pb.ObjectType{
			Code:        "runm.partition",
			Description: "A division of resources. A deployment unit for runm",
		},
		&pb.ObjectType{
			Code:        "runm.image",
			Description: "A bootable bunch of bits",
		},
		&pb.ObjectType{
			Code:        "runm.provider",
			Description: "A provider of some resources, e.g. a compute node or an SR-IOV NIC",
		},
		&pb.ObjectType{
			Code:        "runm.provider_group",
			Description: "A group of providers",
		},
		&pb.ObjectType{
			Code:        "runm.machine",
			Description: "Created by a user, a machine consumes compute resources from one of more providers",
		},
	}
)

// ensureObjectTypes is responsible for making sure etcd has the well-known
// runm object types in storage.
func (s *Store) ensureObjectTypes() error {
	ctx, cancel := s.requestCtx()
	defer cancel()

	s.log.L3("ensuring well-known object types...")

	resp, err := s.kv.Get(
		ctx,
		_OBJECT_TYPES_KEY,
		etcd.WithPrefix(),
		etcd.WithKeysOnly(),
	)
	if err != nil {
		s.log.ERR("error listing object types: %v", err)
		return err
	}
	all := make(map[string]bool, 0)
	for _, k := range resp.Kvs {
		all[string(k.Key)] = true
	}

	for _, ot := range runmObjectTypes {
		if _, ok := all[ot.Code]; !ok {
			s.log.L3("object type %s not in storage. adding...", ot.Code)
			if err = s.objectTypeCreate(ot); err != nil {
				if err == errors.ErrGenerationConflict {
					// some other thread created the object type... just ignore
					continue
				}
				return err
			}
			s.log.L2("created object type %s", ot.Code)
		}
	}
	return nil
}

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

// ObjectTypeCreate writes the supplied ObjectType object to the key at
// $ROOT/object-types/{object_type_code}
func (s *Store) objectTypeCreate(
	obj *pb.ObjectType,
) error {
	ctx, cancel := s.requestCtx()
	defer cancel()

	key := _OBJECT_TYPES_KEY + obj.Code
	value, err := proto.Marshal(obj)
	if err != nil {
		return err
	}
	// create the object type using a transaction that ensures another thread
	// hasn't created an object type with the same key underneath us
	onSuccess := etcd.OpPut(key, string(value))
	// Ensure the key doesn't yet exist
	compare := etcd.Compare(etcd.Version(key), "=", 0)
	resp, err := s.kv.Txn(ctx).If(compare).Then(onSuccess).Commit()

	if err != nil {
		s.log.ERR("failed to create txn in etcd: %v", err)
		return err
	} else if resp.Succeeded == false {
		s.log.L3("another thread already created key %s", key)
		return errors.ErrGenerationConflict
	}
	return nil
}

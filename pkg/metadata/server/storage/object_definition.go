package storage

import (
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/gogo/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
)

const (
	// The primary key index of object definitions
	_OBJECT_DEFINITIONS_BY_TYPE_KEY = "definitions/by-type/"
)

// ObjectDefinitionGet returns an object definition given a partition UUID and
// object type code
func (s *Store) ObjectDefinitionGet(
	partition string,
	objType string,
) (*pb.ObjectDefinition, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	pk := _PARTITIONS_KEY + partition + "/" +
		_OBJECT_DEFINITIONS_BY_TYPE_KEY + objType
	resp, err := s.kv.Get(ctx, pk)
	if err != nil {
		s.log.ERR("error listing object definitions: %v", err)
		return nil, err
	}
	if resp.Count == 0 {
		return nil, errors.ErrNotFound
	}

	obj := &pb.ObjectDefinition{}
	if err = proto.Unmarshal(resp.Kvs[0].Value, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// ObjectDefinitionCreate writes a object definition to backend storage
func (s *Store) ObjectDefinitionCreate(
	pdwr *types.ObjectDefinitionWithReferences,
) (*types.ObjectDefinitionWithReferences, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	pk := _PARTITIONS_KEY + pdwr.Partition.Uuid + "/" +
		_OBJECT_DEFINITIONS_BY_TYPE_KEY + pdwr.ObjectType.Code

	value, err := proto.Marshal(pdwr.Definition)
	if err != nil {
		return nil, err
	}

	// create the object definition using a transaction that ensures another
	// thread hasn't created a object definition with the same key underneath
	// us
	onSuccess := []etcd.Op{
		etcd.OpPut(pk, string(value)),
	}
	// Ensure the key doesn't yet exist
	compare := etcd.Compare(etcd.Version(pk), "=", 0)
	resp, err := s.kv.Txn(ctx).If(compare).Then(onSuccess...).Commit()

	if err != nil {
		s.log.ERR("failed to create txn in etcd: %v", err)
		return nil, err
	} else if resp.Succeeded == false {
		s.log.L3("another thread already created key %s.", pk)
		return nil, errors.ErrGenerationConflict
	}
	return pdwr, nil
}

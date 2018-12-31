package storage

import (
	"fmt"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/gogo/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/conditions"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
	"github.com/runmachine-io/runmachine/pkg/util"
)

const (
	// The primary key index of property definitions
	_PROPERTY_DEFINITIONS_BY_UUID_KEY = "property-definitions/by-uuid/"
)

// PropertyDefinitionDelete removes a property definition from storage
func (s *Store) PropertyDefinitionDelete(
	pdwr *types.PropertyDefinitionWithReferences,
) error {
	ctx, cancel := s.requestCtx()
	defer cancel()

	pk := _PROPERTY_DEFINITIONS_BY_UUID_KEY + pdwr.Definition.Uuid

	resp, err := s.kv.Txn(ctx).Then(etcd.OpDelete(pk)).Commit()

	if err != nil {
		s.log.ERR("failed to create txn in etcd: %v", err)
		return errors.ErrUnknown
	} else if resp.Succeeded == false {
		s.log.ERR("txn commit failed in etcd")
		return errors.ErrUnknown
	}
	return nil
}

// PropertyDefinitionList returns a slice of property definitions matching any
// of a set of supplied filters.
func (s *Store) PropertyDefinitionList(
	any []*conditions.PropertyDefinitionCondition,
) ([]*pb.PropertyDefinition, error) {
	// Each filter is evaluated in an OR fashion, so we keep a hashmap of
	// property definition keys in order to return unique results
	objs := make(map[string]*pb.PropertyDefinition, 0)
	for _, filter := range any {
		filterObjs, err := s.propertyDefinitionsGetMatching(filter)
		if err != nil {
			return nil, err
		}
		for _, obj := range filterObjs {
			objs[obj.Uuid] = obj
		}
	}
	res := make([]*pb.PropertyDefinition, len(objs))
	x := 0
	for _, obj := range objs {
		res[x] = obj
		x += 1
	}
	return res, nil
}

// PropertyDefinitionListWithReferences returns a slice of pointers to
// PropertyDefinitionWithReference structs that have had Partition and
// ObjectType relations expanded inline.
func (s *Store) PropertyDefinitionListWithReferences(
	any []*conditions.PropertyDefinitionCondition,
) ([]*types.PropertyDefinitionWithReferences, error) {
	objects, err := s.PropertyDefinitionList(any)
	if err != nil {
		return nil, err
	}

	// We have two maps for Partition and ObjectType messages that we fetch by
	// partition UUID or object type code while iterating over the objects to
	// delete. These Partition and ObjectType messages are included in the
	// PropertyDefinitionWithReferences structs that are passed around.
	partitions := make(map[string]*pb.Partition, 0)
	objTypes := make(map[string]*pb.ObjectType, 0)
	res := make([]*types.PropertyDefinitionWithReferences, len(objects))
	for x, obj := range objects {
		part, ok := partitions[obj.Partition]
		if !ok {
			part, err = s.partitionGetByUuid(obj.Partition)
			if err != nil {
				msg := fmt.Sprintf(
					"failed to find partition %s while attempting to delete "+
						"property definition with property key %s of object "+
						"type %s",
					obj.Partition,
					obj.Key,
					obj.ObjectType,
				)
				s.log.ERR(msg)
				return nil, errors.ErrPartitionNotFound(obj.Partition)
			}
		}
		ot, ok := objTypes[obj.ObjectType]
		if !ok {
			ot, err = s.ObjectTypeGet(obj.ObjectType)
			if err != nil {
				msg := fmt.Sprintf(
					"failed to find object type %s while attempting to delete "+
						"property definition with property key %s in partition %s",
					obj.ObjectType,
					obj.Key,
					obj.Partition,
				)
				s.log.ERR(msg)
				return nil, errors.ErrObjectTypeNotFound(obj.ObjectType)
			}
		}
		owr := &types.PropertyDefinitionWithReferences{
			Partition:  part,
			ObjectType: ot,
			Definition: obj,
		}
		res[x] = owr
	}
	return res, nil
}

// propertyDefinitionsGetMatching evaluates a single supplied matcher against
// the known property definitions. Returned property definitions are sorted by
// UUID.
func (s *Store) propertyDefinitionsGetMatching(
	matcher types.PropertyDefinitionMatcher,
) ([]*pb.PropertyDefinition, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	opts := []etcd.OpOption{
		etcd.WithSort(etcd.SortByKey, etcd.SortAscend),
		etcd.WithPrefix(),
	}

	resp, err := s.kv.Get(ctx, _PROPERTY_DEFINITIONS_BY_UUID_KEY, opts...)
	if err != nil {
		s.log.ERR("error listing property definitions: %v", err)
		return nil, err
	}
	if resp.Count == 0 {
		return []*pb.PropertyDefinition{}, nil
	}

	res := make([]*pb.PropertyDefinition, resp.Count)
	x := int64(0)
	for _, kv := range resp.Kvs {
		obj := &pb.PropertyDefinition{}
		if err = proto.Unmarshal(kv.Value, obj); err != nil {
			return nil, err
		}
		if !matcher.Matches(obj) {
			continue
		}
		res[x] = obj
		x += 1
	}
	// We return res[:x] here because the above loop may have filtered out some
	// records and we don't want to return "empty slice slots"...
	return res[:x], nil
}

// PropertyDefinitionCreate writes a property definition to backend storage
func (s *Store) PropertyDefinitionCreate(
	pdwr *types.PropertyDefinitionWithReferences,
) (*types.PropertyDefinitionWithReferences, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	if pdwr.Definition.Uuid == "" {
		pdwr.Definition.Uuid = util.NewNormalizedUuid()
	} else {
		pdwr.Definition.Uuid = util.NormalizeUuid(pdwr.Definition.Uuid)
	}

	pk := _PROPERTY_DEFINITIONS_BY_UUID_KEY + pdwr.Definition.Uuid

	value, err := proto.Marshal(pdwr.Definition)
	if err != nil {
		return nil, err
	}

	// create the property definition using a transaction that ensures another
	// thread hasn't created a property definition with the same key underneath
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

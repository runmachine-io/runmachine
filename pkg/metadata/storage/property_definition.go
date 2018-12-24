package storage

import (
	"fmt"
	"strings"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/gogo/protobuf/proto"

	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/metadata/types"
	"github.com/runmachine-io/runmachine/pkg/util"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	// The primary key index of property definitions
	_PROPERTY_DEFINITIONS_BY_UUID_KEY = "property-definitions/by-uuid/"
	// A per-partition index of property definitions by type. The full key
	// includes the object type followed by a slash and the property
	// definition's UUID, which is used as the lookup into the primary key by
	// UUID. This allows us to accomodate multiple property definitions for a
	// single object type within a partition -- since property definitions may
	// be specified for a particular project.
	_PROPERTY_DEFINITIONS_BY_TYPE_KEY = "property-definitions/by-type/"
)

// PropertyDefinitionDelete removes a property definition from storage, removes
// any indexes that may have been created for it and triggers a recalculation
// of the object type's schema
func (s *Store) PropertyDefinitionDelete(
	pdwr *types.PropertyDefinitionWithReferences,
) error {
	ctx, cancel := s.requestCtx()
	defer cancel()

	pk := _PROPERTY_DEFINITIONS_BY_UUID_KEY + pdwr.Definition.Uuid
	byTypeKey := _PARTITIONS_KEY + pdwr.Partition.Uuid + "/" +
		_PROPERTY_DEFINITIONS_BY_TYPE_KEY + pdwr.Type.Code + "/" +
		pdwr.Definition.Uuid

	// creates all the indexes and the objects/by-uuid/ entry using a
	// transaction that ensures if another thread modified anything underneath
	// us, we return an error
	then := []etcd.Op{
		// Delete the primary entry for the property definition
		etcd.OpDelete(pk),
		// Delete the index by type in the partition
		etcd.OpDelete(byTypeKey),
	}
	// TODO(jaypipes): Should we put some If(...) clause in here that verifies
	// the property definition key existed? Not sure it's worth it, really...
	resp, err := s.kv.Txn(ctx).Then(then...).Commit()

	if err != nil {
		s.log.ERR("failed to create txn in etcd: %v", err)
		return errors.ErrUnknown
	} else if resp.Succeeded == false {
		s.log.ERR("txn commit failed in etcd")
		return errors.ErrUnknown
	}
	return nil
}

// PropertyDefinitionGetByPK returns a property definition by partition UUID,
// object type and property key.
func (s *Store) PropertyDefinitionGetByUuid(
	uuid string,
) (*pb.PropertyDefinition, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	pk := _PROPERTY_DEFINITIONS_BY_UUID_KEY + util.NormalizeUuid(uuid)

	gr, err := s.kv.Get(ctx, pk, etcd.WithPrefix())
	if err != nil {
		s.log.ERR("error getting key %s: %v", pk, err)
		return nil, err
	}
	if gr.Count == 0 {
		return nil, errors.ErrNotFound
	} else if gr.Count > 1 {
		return nil, errors.ErrMultipleRecords
	}

	obj := &pb.PropertyDefinition{}
	if err = proto.Unmarshal(gr.Kvs[0].Value, obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// PropertyDefinitionList returns a slice of pointers to PropertyDefinition protobuffer
// messages matching a set of supplied filters.
func (s *Store) PropertyDefinitionList(
	any []*types.PropertyDefinitionFilter,
) ([]*pb.PropertyDefinition, error) {
	// Each filter is evaluated in an OR fashion, so we keep a hashmap of
	// property definition keys in order to return unique results
	objs := make(map[string]*pb.PropertyDefinition, 0)
	for _, filter := range any {
		filterObjs, err := s.propertyDefinitionsGetByFilter(filter)
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
	any []*types.PropertyDefinitionFilter,
) ([]*types.PropertyDefinitionWithReferences, error) {
	objects, err := s.PropertyDefinitionList(any)
	if err != nil {
		return nil, err
	}

	// NOTE(jaypipes): store.PropertyDefinitionDelete() accepts a single
	// argument of type PropertyDefinitionWithReferences. Here, we have two
	// maps for Partition and ObjectType messages that we fetch by partition
	// UUID or object type code while iterating over the objects to delete.
	// These Partition and ObjectType messages are included in the
	// PropertyDefinitionWithReferences structs that are passed to
	// PropertyDefinitionDelete. Gotta love implementing joins in
	// Golang/memory... :(
	partitions := make(map[string]*pb.Partition, 0)
	objTypes := make(map[string]*pb.ObjectType, 0)
	res := make([]*types.PropertyDefinitionWithReferences, len(objects))
	for x, obj := range objects {
		part, ok := partitions[obj.Partition]
		if !ok {
			part, err = s.PartitionGet(obj.Partition)
			if err != nil {
				msg := fmt.Sprintf(
					"failed to find partition %s while attempting to delete "+
						"property definition with property key %s of object "+
						"type %s",
					obj.Partition,
					obj.Key,
					obj.Type,
				)
				s.log.ERR(msg)
				return nil, errors.ErrPartitionNotFound(obj.Partition)
			}
		}
		ot, ok := objTypes[obj.Type]
		if !ok {
			ot, err = s.ObjectTypeGet(obj.Type)
			if err != nil {
				msg := fmt.Sprintf(
					"failed to find object type %s while attempting to delete "+
						"property definition with property key %s in partition %s",
					obj.Type,
					obj.Key,
					obj.Partition,
				)
				s.log.ERR(msg)
				return nil, errors.ErrObjectTypeNotFound(obj.Type)
			}
		}
		owr := &types.PropertyDefinitionWithReferences{
			Partition:  part,
			Type:       ot,
			Definition: obj,
		}
		res[x] = owr
	}
	return res, nil
}

// propertyDefinitionsGetByFilter evaluates a single supplied
// PropertyDefinitionFilter that has been populated with a valid Partition,
// ObjectType and property key to filter by
func (s *Store) propertyDefinitionsGetByFilter(
	filter *types.PropertyDefinitionFilter,
) ([]*pb.PropertyDefinition, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	opts := []etcd.OpOption{
		// TODO(jaypipes): Factor the sorting/limiting/pagination out into a
		// separate utility
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
		if filter.Uuid != "" {
			if filter.Uuid != obj.Uuid {
				continue
			}
		}
		if filter.Partition != nil {
			if filter.Partition.Uuid != obj.Partition {
				continue
			}
		}
		if filter.Type != nil {
			if filter.Type.Code != obj.Type {
				continue
			}
		}
		if filter.Key != "" {
			if filter.UsePrefix {
				if !strings.HasPrefix(obj.Key, filter.Key) {
					continue
				}
			} else {
				if obj.Key != filter.Key {
					continue
				}
			}
		}
		res[x] = obj
		x += 1
	}
	// We return res[:x] here because the above loop may have filtered out some
	// records and we don't want to return "empty slice slots"...
	return res[:x], nil
}

// PropertyDefinitionCreate writes a PropertyDefinition object to the primary
// index and creates all necessary secondary indexes inside the appropriate
// partition key namespace.
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
	byTypeKey := _PARTITIONS_KEY + pdwr.Partition.Uuid + "/" +
		_PROPERTY_DEFINITIONS_BY_TYPE_KEY + pdwr.Type.Code + "/" +
		pdwr.Definition.Uuid

	value, err := proto.Marshal(pdwr.Definition)
	if err != nil {
		return nil, err
	}

	// create the property definition using a transaction that ensures another
	// thread hasn't created a property definition with the same key underneath
	// us
	onSuccess := []etcd.Op{
		etcd.OpPut(pk, string(value)),
		etcd.OpPut(byTypeKey, pdwr.Definition.Uuid),
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

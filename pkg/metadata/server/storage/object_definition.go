package storage

import (
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/gogo/protobuf/proto"

	apitypes "github.com/runmachine-io/runmachine/pkg/api/types"
	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/runmachine-io/runmachine/pkg/util"
)

const (
	// The primary key index of object definitions
	_OBJECT_DEFINITIONS_BY_UUID_KEY = "definitions/by-uuid/"
	// A key namespace by object type
	_OBJECT_DEFINITIONS_BY_TYPE_KEY = "definitions/by-type/"
)

func providerDefinitionKey(
	partUuid string,
	provType string,
) string {
	if partUuid == "" {
		if provType == "" {
			// The global default provider definition
			return _OBJECT_DEFINITIONS_BY_TYPE_KEY + "runm.provider/default"
		}
		// The global default provider definition for a particular provider
		// type
		return _OBJECT_DEFINITIONS_BY_TYPE_KEY + "runm.provider/by-type/" +
			provType
	} else {
		if provType == "" {
			// The provider definition default override for the partition
			return _PARTITIONS_KEY + partUuid + "/" +
				_OBJECT_DEFINITIONS_BY_TYPE_KEY + "runm.provider/default"
		} else {
			// The provider definition override for the partition and specific
			// provider type
			return _PARTITIONS_KEY + partUuid + "/" +
				_OBJECT_DEFINITIONS_BY_TYPE_KEY + "runm.provider/by-type/" +
				provType
		}
	}
}

// ensureDefaultProviderDefinition looks up the global default provider
// definition and if not found, creates it
func (s *Store) ensureDefaultProviderDefinition() error {

	s.log.L3("ensuring default provider definition...")

	if _, err := s.ProviderDefinitionGet("", ""); err != nil {
		if err == errors.ErrNotFound {
			s.log.L3("default provider definition does not exist. creating...")
			pdef := apitypes.DefaultProviderDefinition()
			odef := &pb.ObjectDefinition{
				Schema:              pdef.JSONSchemaString(),
				PropertyPermissions: []*pb.PropertyPermissions{},
			}

			err := s.ProviderDefinitionSet("", "", odef)
			if err != nil {
				s.log.ERR("failed ensuring default provider definition: %s", err)
				return err
			}
			s.log.L1("default provider definition created")
			return nil
		}
		s.log.ERR("failed ensuring default provider definition: %s", err)
		return err
	}
	s.log.L3("default provider definition exists")
	return nil
}

// objectDefinitionGetUuidFromKey returns an object definition UUID given a
// a string key where a UUID is expected
func (s *Store) objectDefinitionGetUuidFromKey(
	key string,
) (string, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	resp, err := s.kv.Get(ctx, key)
	if err != nil {
		s.log.ERR("error getting UUID at key %s: %v", key, err)
		return "", err
	}
	if resp.Count == 0 {
		return "", errors.ErrNotFound
	}

	return string(resp.Kvs[0].Value), nil
}

// ProviderDefinitionGet returns an object definition given an partition UUID
// and provider type. If the partition UUID is empty, returns the global
// default object definition for that provider type. If the provider type is
// empty, returns the global default or partition default for providers.
func (s *Store) ProviderDefinitionGet(
	partUuid string,
	provType string,
) (*pb.ObjectDefinition, error) {
	key := providerDefinitionKey(partUuid, provType)
	uuid, err := s.objectDefinitionGetUuidFromKey(key)
	if err != nil {
		return nil, err
	}
	return s.objectDefinitionGetByUuid(uuid)
}

// ObjectDefinitionGetByUuid returns an object definition given a UUID.
func (s *Store) objectDefinitionGetByUuid(
	uuid string,
) (*pb.ObjectDefinition, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	key := _OBJECT_DEFINITIONS_BY_UUID_KEY + uuid

	resp, err := s.kv.Get(ctx, key)
	if err != nil {
		s.log.ERR("error getting object definition at key %s: %v", key, err)
		return nil, err
	}
	if resp.Count == 0 {
		return nil, errors.ErrNotFound
	}

	var obj pb.ObjectDefinition
	if err = proto.Unmarshal(resp.Kvs[0].Value, &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// ProviderDefinitionSet replaces an object definition in backend storage for
// an optional partition UUID and optional provider type. If the supplied
// partition UUID is empty, this method replaces the default object definition
// for the specified provider type. If the provider type is empty, this method
// replaces the global default provider definition or the partition override
// provider definition.
func (s *Store) ProviderDefinitionSet(
	partUuid string,
	provType string,
	def *pb.ObjectDefinition,
) error {
	ctx, cancel := s.requestCtx()
	defer cancel()

	if def.Uuid == "" {
		def.Uuid = util.NewNormalizedUuid()
	} else {
		def.Uuid = util.NormalizeUuid(def.Uuid)
	}

	value, err := proto.Marshal(def)
	if err != nil {
		return err
	}

	provDefKey := providerDefinitionKey(partUuid, provType)
	uuidKey := _OBJECT_DEFINITIONS_BY_UUID_KEY + def.Uuid

	// create the object definition using a transaction that ensures another
	// thread hasn't created a object definition with the same key underneath
	// us
	onSuccess := []etcd.Op{
		etcd.OpPut(provDefKey, def.Uuid),
		etcd.OpPut(uuidKey, string(value)),
	}
	// TODO(jaypipes): Add in versioning check here
	// compare := etcd.Compare(etcd.Version(uuidKey), "=", cmpVersion)
	resp, err := s.kv.Txn(ctx).Then(onSuccess...).Commit()

	if err != nil {
		s.log.ERR("failed to create txn in etcd: %v", err)
		return err
	} else if resp.Succeeded == false {
		s.log.L3("another thread already created key %s.", provDefKey)
		return errors.ErrDuplicate
	}
	return nil
}

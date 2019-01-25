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

func objectDefinitionKey(
	objType string,
	partUuid string,
) string {
	if partUuid == "" {
		return _OBJECT_DEFINITIONS_BY_TYPE_KEY + objType + "/default"
	}
	return _PARTITIONS_KEY + partUuid + "/" +
		_OBJECT_DEFINITIONS_BY_TYPE_KEY + objType + "/default"
}

// ensureDefaultProviderDefinition looks up the global default provider
// definition and if not found, creates it
func (s *Store) ensureDefaultProviderDefinition() error {

	s.log.L3("ensuring default provider definition...")

	if _, err := s.ObjectDefinitionGet("runm.provider", ""); err != nil {
		if err == errors.ErrNotFound {
			s.log.L3("default provider definition does not exist. creating...")
			pdef := apitypes.DefaultProviderDefinition()
			odef := &pb.ObjectDefinition{
				Schema:              pdef.JSONSchemaString(),
				PropertyPermissions: []*pb.PropertyPermissions{},
			}

			err := s.ObjectDefinitionCreate("runm.provider", "", odef)
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
// an object type and optional partition UUID
func (s *Store) objectDefinitionGetUuidFromKey(
	objType string,
	partUuid string,
) (string, error) {
	ctx, cancel := s.requestCtx()
	defer cancel()

	key := objectDefinitionKey(objType, partUuid)

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

// ObjectDefinitionGet returns an object definition given an
// object type and partition UUID. If the partition UUID is empty, returns the
// global default object definition for that object type
func (s *Store) ObjectDefinitionGet(
	objType string,
	partUuid string,
) (*pb.ObjectDefinition, error) {
	uuid, err := s.objectDefinitionGetUuidFromKey(objType, partUuid)
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

// ObjectDefinitionCreate writes an object definition to backend storage for a
// specified object type and (optional) partition UUID. If the supplid
// partition UUID is empty, this method creates the default object definition
// for that object type.
func (s *Store) ObjectDefinitionCreate(
	objType string,
	partUuid string,
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

	typeKey := objectDefinitionKey(objType, partUuid)
	uuidKey := _OBJECT_DEFINITIONS_BY_UUID_KEY + def.Uuid

	// create the object definition using a transaction that ensures another
	// thread hasn't created a object definition with the same key underneath
	// us
	onSuccess := []etcd.Op{
		etcd.OpPut(typeKey, def.Uuid),
		etcd.OpPut(uuidKey, string(value)),
	}
	// Ensure the key doesn't yet exist
	compare := etcd.Compare(etcd.Version(typeKey), "=", 0)
	resp, err := s.kv.Txn(ctx).If(compare).Then(onSuccess...).Commit()

	if err != nil {
		s.log.ERR("failed to create txn in etcd: %v", err)
		return err
	} else if resp.Succeeded == false {
		s.log.L3("another thread already created key %s.", typeKey)
		return errors.ErrDuplicate
	}
	return nil
}

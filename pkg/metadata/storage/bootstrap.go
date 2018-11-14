package storage

import (
	"context"
	"fmt"

	etcd "github.com/coreos/etcd/clientv3"
	etcd_namespace "github.com/coreos/etcd/clientv3/namespace"
	"github.com/gogo/protobuf/proto"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	// The one-time-use bootstrap token is stored here
	_BOOTSTRAP_KEY = "bootstrap"
	// The index into partition UUIDs by name
	_PARTITIONS_BY_NAME_KEY = "partitions/by-name/%s"
	// The index into Partition protobuffer objects by UUID
	// $PARTITION refers to the key namespace at
	// $ROOT/partitions/by-uuid/{partition_uuid}
	_PARTITIONS_BY_UUID_KEY = "partitions/by-uuid/%s/"
)

var (
	ErrBootstrapFailed = fmt.Errorf("Bootstrap failed.")
)

// init is responsible for creating the etcd key namespace layout that the
// `runm-metadata` service uses to store and fetch information about the
// objects in the system.
func (s *Store) init(ctx context.Context) error {
	// ensure that we have the $ROOT key namespace created
	rootNs := "/"
	rootExists, err := s.keyNamespaceExists(ctx, s.kv, rootNs)
	if err != nil {
		return err
	} else if !rootExists {
		s.log.L3("init: root namespace does not exist. creating...")
		if err = s.keyNamespaceCreate(ctx, s.kv, rootNs); err != nil {
			return err
		}
		s.log.L3("init: root namespace created")
	} else {
		s.log.L3("init: root namespace already exists")
	}
	return nil
}

func (s *Store) kvPartition(partition string) etcd.KV {
	key := fmt.Sprintf(_PARTITIONS_BY_UUID_KEY, partition)
	return etcd_namespace.NewKV(s.kv, key)
}

func (s *Store) keyNamespaceExists(
	ctx context.Context,
	kv etcd.KV,
	ns string,
) (bool, error) {
	gr, err := kv.Get(ctx, ns, etcd.WithPrefix())
	if err != nil {
		s.log.ERR("error getting key namespace %s: %v", ns, err)
		return false, err
	}
	return (len(gr.Kvs) > 0), nil
}

// createKeyNamespace creates the supplied key namespace. If the namespace
// already exists, or another thread creates it during execution, returns nil
func (s *Store) keyNamespaceCreate(
	ctx context.Context,
	kv etcd.KV,
	ns string,
) error {
	// create the key namespace using a transaction that ensures if another
	// thread creates it underneath us, that we just ignore and return nil
	onSuccess := etcd.OpPut(ns, _NO_VALUE)
	// Ensure the key doesn't yet exist
	compare := etcd.Compare(etcd.Version(ns), "=", 0)
	resp, err := kv.Txn(ctx).If(compare).Then(onSuccess).Commit()

	if err != nil {
		s.log.ERR("failed to create txn in etcd: %v", err)
		return err
	} else if resp.Succeeded == false {
		s.log.L3("another thread already created namespace %s. ignoring...", ns)
	}
	return nil
}

// keyHasValue returns true if the supplied key exists and has the supplied
// value, false otherwise
func (s *Store) keyHasValue(
	ctx context.Context,
	kv etcd.KV,
	key string,
	value string,
) bool {
	gr, err := kv.Get(ctx, key)
	if err != nil {
		s.log.ERR("error getting key %s: %v", key, err)
		return false
	}
	if len(gr.Kvs) != 1 {
		return false
	}
	return string(gr.Kvs[0].Value) == value
}

// Bootstrap allows unauthenticated users to create a partition in the
// runm-metadata service if they know the value of a one-time bootstrap token
func (s *Store) Bootstrap(
	token string,
	partName string,
	partUuid string,
) error {
	ctx, cancel := s.requestCtx()
	defer cancel()

	// Do a quick check that the bootstrap token exists and has the same value
	// as the supplied token. If not, log a message and return a generic error.
	if !s.keyHasValue(ctx, s.kv, _BOOTSTRAP_KEY, token) {
		s.log.ERR("bootstrap: bootstrap key does not exist or wrong value supplied for token")
		return ErrBootstrapFailed
	}

	partByNameKey := fmt.Sprintf(_PARTITIONS_BY_NAME_KEY, partName)
	partByUuidKey := fmt.Sprintf(_PARTITIONS_BY_UUID_KEY, partUuid)

	partValue := proto.MarshalTextString(
		&pb.Partition{
			Name: partName,
			Uuid: partUuid,
		},
	)

	// creates the partition keys and deletes the one-time bootstrap token
	// using a transaction that ensures if another thread modified anything
	// underneath us, we return an error
	then := []etcd.Op{
		// Add the entry for the index by partition name
		etcd.OpPut(partByNameKey, partUuid),
		// Add the entry for the index by partition UUID
		etcd.OpPut(partByNameKey, partValue),
		// And remove the one-time-use bootstrap token/key
		etcd.OpDelete(_BOOTSTRAP_KEY),
	}
	compare := []etcd.Cmp{
		// Ensure the partition value and index by name don't yet exist
		etcd.Compare(etcd.Version(partByNameKey), "=", 0),
		etcd.Compare(etcd.Version(partByUuidKey), "=", 0),
		// Ensure the bootstrap token exists and is what we supplied
		etcd.Compare(etcd.Value(_BOOTSTRAP_KEY), "=", token),
	}
	resp, err := s.kv.Txn(ctx).If(compare...).Then(then...).Commit()

	if err != nil {
		s.log.ERR("bootstrap: failed to create txn in etcd: %v", err)
		return ErrBootstrapFailed
	} else if resp.Succeeded == false {
		s.log.L3("bootstrap: another thread bootstrapped")
		return ErrBootstrapFailed
	}
	return nil
}

package storage

import (
	"context"
	"fmt"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/gogo/protobuf/proto"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	// The one-time-use bootstrap token is stored here
	_BOOTSTRAP_KEY = "bootstrap"
)

var (
	ErrBootstrapFailed = fmt.Errorf("Bootstrap failed.")
)

// ensureBootstrap is responsible for creating the one-time-use bootstrap token
// if necessary.
func (s *Store) ensureBootstrap() error {
	ctx, cancel := s.requestCtx()
	defer cancel()
	if s.cfg.BootstrapToken == "" {
		s.log.L3("no bootstrap token specified. ensuring bootstrap token does not exist...")
		if _, err := s.kv.Delete(ctx, _BOOTSTRAP_KEY); err != nil {
			s.log.ERR("failed trying to delete the bootstrap key: %s", err)
			return err
		}
	} else {
		s.log.L3("ensuring bootstrap token...")
		if _, err := s.kv.Put(ctx, _BOOTSTRAP_KEY, s.cfg.BootstrapToken); err != nil {
			s.log.ERR("failed trying to create the bootstrap key: %s", err)
			return err
		}
		s.log.L2("bootstrap token created")
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

	partByNameKey := _PARTITIONS_BY_NAME_KEY + partName
	partByUuidKey := _PARTITIONS_BY_UUID_KEY + partUuid

	partValue, err := proto.Marshal(
		&pb.Partition{
			Name: partName,
			Uuid: partUuid,
		},
	)
	if err != nil {
		s.log.ERR("bootstrap: failed to serialize object: %v", err)
		return ErrBootstrapFailed
	}

	// creates the partition keys and deletes the one-time bootstrap token
	// using a transaction that ensures if another thread modified anything
	// underneath us, we return an error
	then := []etcd.Op{
		// Add the entry for the index by partition name
		etcd.OpPut(partByNameKey, partUuid),
		// Add the entry for the index by partition UUID
		etcd.OpPut(partByUuidKey, string(partValue)),
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

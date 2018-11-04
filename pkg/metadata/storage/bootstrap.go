package storage

import (
	"context"

	etcd "github.com/coreos/etcd/clientv3"
)

// bootstrap is responsible for creating the etcd key namespace layout that the
// `runm-metadata` service uses to store and fetch information about the
// objects in the system.
func (s *Store) bootstrap(ctx context.Context) error {
	// ensure that we have the $ROOT key namespace created
	rootNs := "/"
	rootExists, err := s.keyNamespaceExists(ctx, s.kv, rootNs)
	if err != nil {
		return err
	} else if !rootExists {
		s.log.L3("bootstrapping: root namespace does not exist. creating...")
		if err = s.keyNamespaceCreate(ctx, s.kv, rootNs); err != nil {
			return err
		}
		s.log.L3("bootstrapping: root namespace created")
	} else {
		s.log.L3("bootstrapping: root namespace already exists")
	}
	s.bootstrapped = true
	return nil
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

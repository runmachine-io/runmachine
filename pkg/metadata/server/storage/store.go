package storage

import (
	"context"

	etcd "github.com/coreos/etcd/clientv3"
	etcd_namespace "github.com/coreos/etcd/clientv3/namespace"

	"github.com/runmachine-io/runmachine/pkg/etcdutil"
	"github.com/runmachine-io/runmachine/pkg/logging"
	"github.com/runmachine-io/runmachine/pkg/metadata/server/config"
)

const (
	// The key that carves out a namespace for the runm-metadata service to
	// store stuff in etcd. This namespace comes directly UNDER the
	// Config.EtcdKeyPrefix namespace. This namespace is referred to as $ROOT
	_SERVICE_KEY = "runm/metadata/"

	// Used when creating empty leaf-level keys or key namespaces
	_NO_VALUE = ""

	// Used in ranges when limiting searches on UUID indexes
	_MAX_UUID = "ffffffffffffffffffffffffffffffff"
)

type Store struct {
	log    *logging.Logs
	cfg    *config.Config
	client *etcd.Client
	kv     etcd.KV
}

func New(log *logging.Logs, cfg *config.Config) (*Store, error) {
	etcdCfg := cfg.Etcd
	client, err := etcdutil.Connect(log, etcdCfg)
	if err != nil {
		return nil, err
	}
	s := &Store{
		log:    log,
		cfg:    cfg,
		client: client,
		kv:     etcd_namespace.NewKV(client.KV, etcdCfg.KeyPrefix+_SERVICE_KEY),
	}
	if err = s.ensureObjectTypes(); err != nil {
		return nil, err
	}
	if err = s.ensureProviderTypes(); err != nil {
		return nil, err
	}
	if err = s.ensureDefaultProviderDefinition(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) requestCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.Background(),
		s.cfg.Etcd.RequestTimeoutSeconds,
	)
}

package storage

import (
	"context"

	etcd "github.com/coreos/etcd/clientv3"
	etcd_namespace "github.com/coreos/etcd/clientv3/namespace"

	"github.com/runmachine-io/runmachine/pkg/logging"
	"github.com/runmachine-io/runmachine/pkg/metadata/config"
)

const (
	// The key that carves out a namespace for the runm-metadata service to
	// store stuff in etcd. This namespace comes directly UNDER the
	// Config.EtcdKeyPrefix namespace
	_SERVICE_KEY = "runm-metadata/"

	// Used when creating empty leaf-level keys or key namespaces
	_NO_VALUE = ""
)

type Store struct {
	log          *logging.Logs
	cfg          *config.Config
	client       *etcd.Client
	kv           etcd.KV
	bootstrapped bool
}

func New(log *logging.Logs, cfg *config.Config) (*Store, error) {
	client, err := connect(log, cfg)
	if err != nil {
		return nil, err
	}
	s := &Store{
		log:    log,
		cfg:    cfg,
		client: client,
		kv:     etcd_namespace.NewKV(client.KV, cfg.EtcdKeyPrefix+_SERVICE_KEY),
	}
	ctx, cancel := s.requestCtx()
	defer cancel()
	if err = s.bootstrap(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) requestCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.Background(),
		s.cfg.EtcdRequestTimeoutSeconds,
	)
}

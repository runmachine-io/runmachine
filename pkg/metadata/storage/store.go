package storage

import (
	"context"

	etcd "github.com/coreos/etcd/clientv3"
	etcd_namespace "github.com/coreos/etcd/clientv3/namespace"

	"github.com/jaypipes/runmachine/pkg/logging"
	"github.com/jaypipes/runmachine/pkg/metadata/config"
)

const (
	// The key that carves out a namespace for the runm-metadata service to
	// store stuff in etcd. This namespace comes directly UNDER the
	// Config.EtcdKeyPrefix namespace
	_SERVICE_KEY = "runm-metadata/"
)

type Store struct {
	log    *logging.Logs
	cfg    *config.Config
	client *etcd.Client
	kv     etcd.KV
}

func New(log *logging.Logs, cfg *config.Config) (*Store, error) {
	client, err := connect(log, cfg)
	if err != nil {
		return nil, err
	}
	return &Store{
		log:    log,
		cfg:    cfg,
		client: client,
		kv:     etcd_namespace.NewKV(client.KV, cfg.EtcdKeyPrefix+_SERVICE_KEY),
	}, nil
}

func (s *Store) requestCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.Background(),
		s.cfg.EtcdRequestTimeoutSeconds,
	)
}

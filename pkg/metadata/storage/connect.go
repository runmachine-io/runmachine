package storage

import (
	"context"
	"net"
	"syscall"

	"github.com/cenkalti/backoff"
	etcd "github.com/coreos/etcd/clientv3"
	"google.golang.org/grpc"

	"github.com/jaypipes/runmachine/pkg/logging"
	"github.com/jaypipes/runmachine/pkg/metadata/config"
)

// Returns an etcd3 client using an exponential backoff and reconnect strategy.
// This is to be tolerant of the etcd infrastructure VMs/containers starting
// *after* the service that requires it.
func connect(
	log *logging.Logs,
	cfg *config.Config,
) (*etcd.Client, error) {
	var err error
	var client *etcd.Client
	fatal := false
	connectTimeout := cfg.EtcdConnectTimeoutSeconds
	etcdCfg := cfg.EtcdConfig()
	etcdEps := etcdCfg.Endpoints

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = connectTimeout

	log.L2("connecting to etcd endpoints %v (w/ %s overall timeout).",
		etcdEps, connectTimeout.String())

	fn := func() error {
		client, err = etcd.New(*etcdCfg)
		if err != nil {
			if err == grpc.ErrClientConnTimeout ||
				err == context.Canceled ||
				err == context.DeadlineExceeded {
				// Each of these scenarios are errors that we can retry the
				// operation. Services may come up in different order and we
				// don't want to require a specific order of startup...
				return err
			}
			switch t := err.(type) {
			case *net.OpError:
				oerr := err.(*net.OpError)
				if oerr.Temporary() || oerr.Timeout() {
					// Each of these scenarios are errors that we can retry
					// the operation. Services may come up in different
					// order and we don't want to require a specific order
					// of startup...
					return err
				}
				if t.Op == "dial" {
					destAddr := oerr.Addr
					if destAddr == nil {
						// Unknown host... probably a DNS failure and not
						// something we're going to be able to recover from in
						// a retry, so bail out
						fatal = true
					}
					// If not unknown host, most likely a dial: tcp
					// connection refused. In that case, let's retry. etcd
					// may not have been brought up before the calling
					// application/service..
					return err
				} else if t.Op == "read" {
					// connection refused. In that case, let's retry. etcd
					// may not have been brought up before the calling
					// application/service..
					return err
				}
			case syscall.Errno:
				if t == syscall.ECONNREFUSED {
					// connection refused. In that case, let's retry. etcd
					// may not have been brought up before the calling
					// application/service..
					return err
				}
			default:
				log.L2("got unrecoverable %T error: %v attempting to "+
					"connect to etcd", err, err)
				fatal = true
				return err
			}
		}
		return nil
	}

	ticker := backoff.NewTicker(bo)

	attempts := 0
	for _ = range ticker.C {
		if err = fn(); err != nil {
			attempts += 1
			if fatal {
				break
			}
			log.L2("failed to connect to gsr: %v. retrying.", err)
			continue
		}

		ticker.Stop()
		break
	}

	if err != nil {
		log.ERR("failed to connect to gsr. final error reported: %v", err)
		log.L2("attempted %d times over %v. exiting.",
			attempts, bo.GetElapsedTime().String())
		return nil, err
	}
	return client, nil
}

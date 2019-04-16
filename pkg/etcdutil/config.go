package etcdutil

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
)

type Config struct {
	UseTLS                bool
	CertPath              string
	KeyPath               string
	Endpoints             []string
	KeyPrefix             string
	ConnectTimeoutSeconds time.Duration
	RequestTimeoutSeconds time.Duration
	DialTimeoutSeconds    time.Duration
}

// Returns an etcd configuration struct populated with all configured options.
func (c *Config) EtcdConfig() *etcd.Config {
	return &etcd.Config{
		Endpoints:   c.Endpoints,
		DialTimeout: c.DialTimeoutSeconds,
		TLS:         c.TLSConfig(),
	}
}

// Returns the TLS configuration struct to use with etcd client.
func (c *Config) TLSConfig() *tls.Config {
	cfg := &tls.Config{}

	if !c.UseTLS {
		return nil
	}
	certPath := c.CertPath
	keyPath := c.KeyPath

	if certPath == "" || keyPath == "" {
		fmt.Fprintf(
			os.Stderr,
			"error setting up TLS configuration. Either cert or "+
				"key path not specified.",
		)
		return nil
	}

	certContent, err := ioutil.ReadFile(certPath)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"error getting cert content: %v",
			err,
		)
		return nil
	}

	keyContent, err := ioutil.ReadFile(keyPath)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"error getting key content: %v",
			err,
		)
		return nil
	}

	kp, err := tls.X509KeyPair(certContent, keyContent)
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"error setting up TLS cert: %v.",
			err,
		)
		return nil
	}

	cfg.MinVersion = tls.VersionTLS10
	cfg.InsecureSkipVerify = false
	cfg.Certificates = []tls.Certificate{kp}
	return cfg
}

// Returns the set of etcd3 endpoints used by runm
func NormalizeEndpoints(epsStr string) []string {
	eps := strings.Split(epsStr, ",")
	res := make([]string, len(eps))
	// Ensure endpoints begin with http[s]:// and contain a port. If missing,
	// add default etcd port.
	for x, ep := range eps {
		if !strings.HasPrefix(ep, "http") {
			ep = "http://" + ep
		}
		if strings.Count(ep, ":") == 1 {
			ep = ep + ":2379"
		}
		res[x] = ep
	}
	return res
}

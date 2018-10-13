package config

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	flag "github.com/ogier/pflag"

	"github.com/jaypipes/envutil"
	"github.com/jaypipes/runmachine/pkg/util"
)

const (
	cfgPath                          = "/etc/runmachine/metadata"
	defaultUseTLS                    = false
	defaultBindPort                  = 10000
	defaultServiceName               = "runmachine-metadata"
	defaultEtcdEndpoints             = "http://127.0.0.1:2379"
	defaultEtcdKeyPrefix             = "runm-metadata/"
	defaultEtcdConnectTimeoutSeconds = 300
	defaultEtcdRequestTimeoutSeconds = 1
	defaultEtcdDialTimeoutSeconds    = 1
)

var (
	defaultCertPath = filepath.Join(cfgPath, "server.pem")
	defaultKeyPath  = filepath.Join(cfgPath, "server.key")
	defaultBindHost = util.BindHost()
)

type Config struct {
	UseTLS                    bool
	CertPath                  string
	KeyPath                   string
	BindHost                  string
	BindPort                  int
	ServiceName               string
	EtcdEndpoints             []string
	EtcdKeyPrefix             string
	EtcdConnectTimeoutSeconds time.Duration
	EtcdRequestTimeoutSeconds time.Duration
	EtcdDialTimeoutSeconds    time.Duration
}

func ConfigFromOpts() *Config {
	optUseTLS := flag.Bool(
		"use-tls",
		envutil.WithDefaultBool(
			"RUNM_METADATA_USE_TLS", defaultUseTLS,
		),
		"Connection uses TLS if true, else plain TCP",
	)
	optCertPath := flag.String(
		"cert-path",
		envutil.WithDefault(
			"RUNM_METADATA_CERT_PATH", defaultCertPath,
		),
		"Path to the TLS cert file",
	)
	optKeyPath := flag.String(
		"key-path",
		envutil.WithDefault(
			"RUNM_METADATA_KEY_PATH", defaultKeyPath,
		),
		"Path to the TLS key file",
	)
	optHost := flag.String(
		"bind-address",
		envutil.WithDefault(
			"RUNM_METADATA_BIND_HOST", defaultBindHost,
		),
		"The host address the server will listen on",
	)
	optPort := flag.Int(
		"bind-port",
		envutil.WithDefaultInt(
			"RUNM_METADATA_BIND_PORT", defaultBindPort,
		),
		"The port the server will listen on",
	)
	optServiceName := flag.String(
		"service-name",
		envutil.WithDefault(
			"RUNM_METADATA_SERVICE_NAME", defaultServiceName,
		),
		"Name to use when registering with the service registry",
	)

	etcdEndpointsStr := flag.String(
		"storage-etcd-endpoints",
		envutil.WithDefault(
			"RUNM_METADATA_STORAGE_ETCD_ENDPOINTS", defaultEtcdEndpoints,
		),
		"Comma-delimited list of etcd3 endpoints to use for metadata storage",
	)
	endpoints := etcdNormalizeEndpoints(*etcdEndpointsStr)
	keyPrefix := flag.String(
		"storage-etcd-key-prefix",
		strings.TrimRight(
			envutil.WithDefault(
				"RUNM_METADATA_STORAGE_ETCD_KEY_PREFIX",
				defaultEtcdKeyPrefix,
			),
			"/",
		)+"/",
		"Prefix to use to segregate all runm-metadata inside etcd3",
	)
	connectTimeout := flag.Int(
		"storage-etcd-connect-timeout-seconds",
		envutil.WithDefaultInt(
			"RUNM_METADATA_STORAGE_ETCD_CONNECT_TIMEOUT_SECONDS",
			defaultEtcdConnectTimeoutSeconds,
		),
		"Total number of seconds to attempt connection to etcd",
	)
	requestTimeout := flag.Int(
		"storage-etcd-request-timeout-seconds",
		envutil.WithDefaultInt(
			"RUNM_METADATA_STORAGE_ETCD_REQUEST_TIMEOUT_SECONDS",
			defaultEtcdRequestTimeoutSeconds,
		),
		"Number of seconds to timeout attempting each individual etcd request",
	)
	dialTimeout := flag.Int(
		"storage-etcd-dial-timeout-seconds",
		envutil.WithDefaultInt(
			"RUNM_METADATA_STORAGE_ETCD_DIAL_TIMEOUT_SECONDS",
			defaultEtcdDialTimeoutSeconds,
		),
		"Number of seconds to timeout attempting each connect/dial attempt to etcd",
	)

	flag.Parse()

	return &Config{
		UseTLS:                    *optUseTLS,
		CertPath:                  *optCertPath,
		KeyPath:                   *optKeyPath,
		BindHost:                  *optHost,
		BindPort:                  *optPort,
		ServiceName:               *optServiceName,
		EtcdEndpoints:             endpoints,
		EtcdKeyPrefix:             *keyPrefix,
		EtcdConnectTimeoutSeconds: time.Duration(*connectTimeout) * time.Second,
		EtcdRequestTimeoutSeconds: time.Duration(*requestTimeout) * time.Second,
		EtcdDialTimeoutSeconds:    time.Duration(*dialTimeout) * time.Second,
	}
}

// Returns an etcd configuration struct populated with all configured options.
func (c *Config) EtcdConfig() *etcd.Config {
	return &etcd.Config{
		Endpoints:   c.EtcdEndpoints,
		DialTimeout: c.EtcdDialTimeoutSeconds,
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

// Returns the set of etcd3 endpoints used by runm-metadata
func etcdNormalizeEndpoints(epsStr string) []string {
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

package metadata

import (
	"path/filepath"

	flag "github.com/ogier/pflag"

	"github.com/jaypipes/envutil"
	"github.com/jaypipes/runmachine/pkg/util"
)

const (
	cfgPath            = "/etc/runmachine/metadata"
	defaultUseTLS      = false
	defaultBindPort    = 10000
	defaultServiceName = "runmachine-metadata"
)

var (
	defaultCertPath = filepath.Join(cfgPath, "server.pem")
	defaultKeyPath  = filepath.Join(cfgPath, "server.key")
	defaultBindHost = util.BindHost()
)

type Config struct {
	UseTLS      bool
	CertPath    string
	KeyPath     string
	BindHost    string
	BindPort    int
	ServiceName string
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

	flag.Parse()

	return &Config{
		UseTLS:      *optUseTLS,
		CertPath:    *optCertPath,
		KeyPath:     *optKeyPath,
		BindHost:    *optHost,
		BindPort:    *optPort,
		ServiceName: *optServiceName,
	}
}

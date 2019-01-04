package config

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	flag "github.com/ogier/pflag"

	"github.com/jaypipes/envutil"
	"github.com/runmachine-io/runmachine/pkg/util"
)

const (
	cfgPath                    = "/etc/runmachine/api"
	defaultUseTLS              = false
	defaultBindPort            = 10002
	defaultServiceName         = "runmachine-api"
	defaultMetadataServiceName = "runmachine-metadata"
	defaultResourceServiceName = "runmachine-resource"
)

var (
	defaultCertPath = filepath.Join(cfgPath, "server.pem")
	defaultKeyPath  = filepath.Join(cfgPath, "server.key")
	defaultBindHost = util.BindHost()
)

type Config struct {
	UseTLS              bool
	CertPath            string
	KeyPath             string
	BindHost            string
	BindPort            int
	ServiceName         string
	MetadataServiceName string
	ResourceServiceName string
}

func ConfigFromOpts() *Config {
	optUseTLS := flag.Bool(
		"use-tls",
		envutil.WithDefaultBool(
			"RUNM_API_USE_TLS", defaultUseTLS,
		),
		"Connection uses TLS if true, else plain TCP",
	)
	optCertPath := flag.String(
		"cert-path",
		envutil.WithDefault(
			"RUNM_API_CERT_PATH", defaultCertPath,
		),
		"Path to the TLS cert file",
	)
	optKeyPath := flag.String(
		"key-path",
		envutil.WithDefault(
			"RUNM_API_KEY_PATH", defaultKeyPath,
		),
		"Path to the TLS key file",
	)
	optHost := flag.String(
		"bind-address",
		envutil.WithDefault(
			"RUNM_API_BIND_HOST", defaultBindHost,
		),
		"The host address the server will listen on",
	)
	optPort := flag.Int(
		"bind-port",
		envutil.WithDefaultInt(
			"RUNM_API_BIND_PORT", defaultBindPort,
		),
		"The port the server will listen on",
	)
	optServiceName := flag.String(
		"service-name",
		envutil.WithDefault(
			"RUNM_API_SERVICE_NAME", defaultServiceName,
		),
		"Name to use when registering the API service with the service registry",
	)
	optMetadataServiceName := flag.String(
		"metadata-service-name",
		envutil.WithDefault(
			"RUNM_METADATA_SERVICE_NAME", defaultMetadataServiceName,
		),
		"Name to use when querying the service registry for the metadata service",
	)
	optResourceServiceName := flag.String(
		"resource-service-name",
		envutil.WithDefault(
			"RUNM_RESOURCE_SERVICE_NAME", defaultResourceServiceName,
		),
		"Name to use when querying the service registry for the resource service",
	)

	flag.Parse()

	return &Config{
		UseTLS:              *optUseTLS,
		CertPath:            *optCertPath,
		KeyPath:             *optKeyPath,
		BindHost:            *optHost,
		BindPort:            *optPort,
		ServiceName:         *optServiceName,
		MetadataServiceName: *optMetadataServiceName,
		ResourceServiceName: *optResourceServiceName,
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

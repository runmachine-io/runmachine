package main

import (
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/jaypipes/runmachine/pkg/metadata"
	pb "github.com/jaypipes/runmachine/proto"

	"github.com/jaypipes/runmachine/pkg/logging"
)

func main() {
	log := logging.New(logging.ConfigFromOpts())

	defer log.WithSection("runm-metadata")()

	cfg := metadata.ConfigFromOpts()

	md, err := metadata.NewServer(cfg, log)
	if err != nil {
		log.ERR("failed to create runm-metadata server: %v", err)
		os.Exit(1)
	}
	defer srv.Close()

	addr := fmt.Sprintf("%s:%d", cfg.BindHost, cfg.BindPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.ERR("failed to listen: %v", err)
		os.Exit(1)
	}
	log.L2("listening on TCP %s", addr)

	// Set up the gRPC server listening on incoming TCP connections on our port
	var opts []grpc.ServerOption
	if cfg.UseTLS {
		creds, err := credentials.NewServerTLSFromFile(
			cfg.CertPath,
			cfg.KeyPath,
		)
		if err != nil {
			log.ERR("failed to generate credentials: %v", err)
			os.Exit(1)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
		log.L2("using credentials file %v", cfg.KeyPath)
	}
	s := grpc.NewServer(opts...)
	pb.RegisterRunmMetadataServer(s, md)
	s.Serve(lis)
}

package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/runmachine-io/runmachine/pkg/api/server"
	"github.com/runmachine-io/runmachine/pkg/api/server/config"
	pb "github.com/runmachine-io/runmachine/proto"

	"github.com/runmachine-io/runmachine/pkg/logging"
)

func main() {
	log := logging.New(logging.ConfigFromOpts())

	defer log.WithSection("runm-api")()

	cfg := config.ConfigFromOpts()

	md, err := server.New(cfg, log)
	if err != nil {
		log.ERR("failed to create runm-api server: %v", err)
		os.Exit(1)
	}
	defer md.Close()

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

	// Handle SIGTERM signals and close our Service instance, which should take
	// care of notifying the service registry about our endpoint going away
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.L1("received %s.", sig)
		md.Close()
		done <- true
	}()

	s := grpc.NewServer(opts...)
	pb.RegisterRunmAPIServer(s, md)
	s.Serve(lis)
}

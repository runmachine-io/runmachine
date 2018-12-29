package server

import (
	"fmt"
	"os"

	"github.com/jaypipes/gsr"
	"google.golang.org/grpc"

	"github.com/runmachine-io/runmachine/pkg/api/server/config"
	"github.com/runmachine-io/runmachine/pkg/logging"
	metapb "github.com/runmachine-io/runmachine/proto"
)

var (
	ErrBadInput = fmt.Errorf("Bad input. Check response.Errors for more information")
)

type Server struct {
	log        *logging.Logs
	cfg        *config.Config
	registry   *gsr.Registry
	metaclient metapb.RunmMetadataClient
}

func (s *Server) Close() {
	addr := fmt.Sprintf("%s:%d", s.cfg.BindHost, s.cfg.BindPort)
	s.log.L1(
		"unregistering %s:%s endpoint in gsr",
		s.cfg.ServiceName,
		addr,
	)
	ep := &gsr.Endpoint{
		Service: &gsr.Service{Name: s.cfg.ServiceName},
		Address: addr,
	}
	err := s.registry.Unregister(ep)
	if err != nil {
		s.log.ERR("failed to unregister: %s\n", err)
	}
}

func metaConnect(connectHost string, connectPort int) *grpc.ClientConn {
	var opts []grpc.DialOption
	// TODO(jaypipes): Don't hardcode this to WithInsecure
	opts = append(opts, grpc.WithInsecure())
	addr := fmt.Sprintf("%s:%d", connectHost, connectPort)
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return nil
	}
	return conn
}

func New(
	cfg *config.Config,
	log *logging.Logs,
) (*Server, error) {
	log.L3("connecting to gsr service registry.")
	registry, err := gsr.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create gsr.Registry object: %v", err)
	}
	log.L2("connected to gsr service registry.")

	conn := metaConnect("172.17.0.3", 10000)
	metaclient := metapb.NewRunmMetadataClient(conn)

	// Register this runm-metadata service endpoint with the service registry
	addr := fmt.Sprintf("%s:%d", cfg.BindHost, cfg.BindPort)
	ep := gsr.Endpoint{
		Service: &gsr.Service{Name: cfg.ServiceName},
		Address: addr,
	}
	err = registry.Register(&ep)
	if err != nil {
		return nil, fmt.Errorf("failed to register %v with gsr: %v", ep, err)
	}
	log.L2(
		"registered %s service endpoint running at %s with gsr.",
		cfg.ServiceName,
		addr,
	)

	return &Server{
		log:        log,
		cfg:        cfg,
		registry:   registry,
		metaclient: metaclient,
	}, nil
}

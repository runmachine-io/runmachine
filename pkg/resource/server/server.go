package server

import (
	"fmt"

	"github.com/jaypipes/gsr"

	"github.com/runmachine-io/runmachine/pkg/logging"
	"github.com/runmachine-io/runmachine/pkg/resource/server/config"
	"github.com/runmachine-io/runmachine/pkg/resource/server/storage"
	metapb "github.com/runmachine-io/runmachine/proto"
)

var (
	ErrBadInput = fmt.Errorf("Bad input. Check response.Errors for more information")
)

type Server struct {
	log        *logging.Logs
	cfg        *config.Config
	registry   *gsr.Registry
	store      *storage.Store
	metaclient metapb.RunmMetadataClient
}

func (s *Server) Close() {
	addr := fmt.Sprintf("%s:%d", s.cfg.BindHost, s.cfg.BindPort)
	s.log.L3(
		"unregistering %s:%s endpoint in gsr...",
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

	store, err := storage.New(log, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize resource storage: %v", err)
	}
	log.L2("initialized resource storage.")

	// Register this runm-api service endpoint with the service registry
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
		log:      log,
		cfg:      cfg,
		registry: registry,
		store:    store,
	}, nil
}

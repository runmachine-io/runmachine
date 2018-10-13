package metadata

import (
	"fmt"

	"github.com/jaypipes/gsr"

	"github.com/jaypipes/runmachine/pkg/logging"
	"github.com/jaypipes/runmachine/pkg/metadata/config"
	"github.com/jaypipes/runmachine/pkg/metadata/storage"
)

type Server struct {
	log      *logging.Logs
	cfg      *config.Config
	registry *gsr.Registry
	store    *storage.Store
}

func (s *Server) Close() {
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
		return nil, fmt.Errorf("failed to connect to metadata storage: %v", err)
	}
	log.L2("connected to metadata storage.")

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
		log:      log,
		cfg:      cfg,
		registry: registry,
		store:    store,
	}, nil
}

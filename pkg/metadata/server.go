package metadata

import (
	"fmt"

	"github.com/jaypipes/gsr"

	"github.com/jaypipes/runmachine/pkg/logging"
)

type Server struct {
	log      *logging.Logs
	cfg      *Config
	registry *gsr.Registry
}

func (s *Server) Close() {
}

func NewServer(
	cfg *Config,
	log *logging.Logs,
) (*Server, error) {
	log.L3("connecting to gsr service registry.")
	registry, err := gsr.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create gsr.Registry object: %v", err)
	}
	log.L2("connected to gsr service registry.")

	// Register this runm-metadata service endpoint with the service registry
	addr := fmt.Sprintf("%s:%d", cfg.BindHost, cfg.BindPort)
	ep := gsr.Endpoint{
		Service: &gsr.Service{Name: cfg.ServiceName},
		Address: addr,
	}
	err = registry.Register(&ep)
	if err != nil {
		log.ERR("unable to register %v with gsr: %v", ep, err)
	}
	log.L2(
		"registered %s service endpoint running at %s with gsr.",
		cfg.ServiceName,
		addr,
	)

	s := &Server{
		log:      log,
		cfg:      cfg,
		registry: registry,
	}
	return s, nil
}

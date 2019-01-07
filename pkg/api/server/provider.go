package server

import (
	"context"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/runmachine-io/runmachine/pkg/api/types"
	"github.com/runmachine-io/runmachine/pkg/errors"
	metapb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/runmachine-io/runmachine/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	yaml "gopkg.in/yaml.v2"
)

// ProviderGet looks up a provider by UUID or name and returns a Provider
// protobuf message.
func (s *Server) ProviderGet(
	ctx context.Context,
	req *pb.ProviderGetRequest,
) (*pb.Provider, error) {
	if req.Filter == nil || req.Filter.Search == "" {
		return nil, ErrSearchRequired
	}
	var err error
	var search string
	search = req.Filter.Search
	if !util.IsUuidLike(search) {
		// Look up the provider's UUID in the metadata service by name
		search, err = s.uuidFromName(req.Session, "runm.provider", search)
		if err != nil {
			return nil, err
		}
	}
	p, err := s.providerGetByUuid(req.Session, search)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFound
		}
		s.log.ERR("failed getting provider with UUID %s: %s", search, err)
		return nil, ErrUnknown
	}
	// Grab the object's name from the metadata service
	name, err := s.nameFromUuid(req.Session, search)
	if err != nil {
		if err == errors.ErrNotFound {
			s.log.ERR(
				"DATA CORRUPTION! failed getting name for provider with "+
					"UUID %s: object with UUID %s does not exist in metadata "+
					"service but should exist",
				search, err,
			)
			name = ""
		} else {
			s.log.ERR("failed getting provider with UUID %s: %s", search, err)
			return nil, ErrUnknown
		}
	}
	return &pb.Provider{
		Partition:    p.Partition,
		ProviderType: p.ProviderType,
		Name:         name,
		Uuid:         p.Uuid,
		Generation:   p.Generation,
	}, nil
}

// validateProviderCreateRequest ensures that the data the user sent in the
// request payload can be unmarshal'd properly into YAML, contains all relevant
// fields and meets things like property meta validation checks.
func (s *Server) validateProviderCreateRequest(
	req *pb.CreateRequest,
) (*types.Provider, error) {
	var p types.Provider
	if err := yaml.Unmarshal(req.Payload, &p); err != nil {
		return nil, err
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}

	// Check that the supplied provider type exists
	ptCode := p.ProviderType
	_, err := s.providerTypeGetByCode(req.Session, ptCode)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errProviderTypeNotFound(ptCode)
		}
		s.log.ERR("failed checking provider type: %s", err)
		return nil, ErrUnknown
	}

	return &p, nil
}

func (s *Server) ProviderCreate(
	ctx context.Context,
	req *pb.CreateRequest,
) (*pb.ProviderCreateResponse, error) {
	// TODO(jaypipes): AUTHZ check if user can write objects

	input, err := s.validateProviderCreateRequest(req)
	if err != nil {
		return nil, err
	}

	s.log.L3(
		"creating new provider in partition %s with name %s...",
		input.Partition,
		input.Name,
	)

	// First save the object in the metadata service
	provObj := &metapb.Object{
		Partition:  input.Partition,
		ObjectType: "runm.provider",
		Uuid:       input.Uuid,
		Name:       input.Name,
		Tags:       input.Tags,
	}
	if input.Properties != nil {
		props := make([]*metapb.Property, len(input.Properties))
		for key, val := range input.Properties {
			props = append(props, &metapb.Property{
				Key:   key,
				Value: val,
			})
		}
		provObj.Properties = props
	}
	createdObj, err := s.objectCreate(req.Session, provObj)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			scode := s.Code()
			if scode == codes.FailedPrecondition || scode == codes.NotFound {
				return nil, err
			}
		}
		s.log.ERR(
			"failed creating provider object in metadata service: %s",
			err,
		)
		return nil, ErrUnknown
	}

	input.Uuid = createdObj.Uuid

	// Next save the provider record in the resource service
	resProv, err := s.providerCreate(req.Session, input)
	if err != nil {
		return nil, err
	}
	s.log.L1(
		"created new provider with UUID %s in partition %s with name %s",
		input.Uuid,
		createdObj.Partition,
		input.Name,
	)

	return &pb.ProviderCreateResponse{
		Provider: &pb.Provider{
			Uuid:         input.Uuid,
			Name:         input.Name,
			Partition:    createdObj.Partition,
			ProviderType: input.ProviderType,
			Generation:   resProv.Generation,
		},
	}, nil
}

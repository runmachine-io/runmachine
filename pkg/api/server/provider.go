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
	if req.Filter == nil || req.Filter.PrimaryFilter == nil || req.Filter.PrimaryFilter.Search == "" {
		return nil, ErrSearchRequired
	}
	var err error
	var search string
	search = req.Filter.PrimaryFilter.Search
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
	// Grab the object from the metadata service
	obj, err := s.objectFromUuid(req.Session, search)
	if err != nil {
		if err == errors.ErrNotFound {
			s.log.ERR(
				"DATA CORRUPTION! failed getting object with "+
					"UUID %s: object with UUID %s does not exist in metadata "+
					"service but providerGetByUuid returned a provider",
				search, err,
			)
			return nil, ErrUnknown
		} else {
			s.log.ERR("failed getting object with UUID %s: %s", search, err)
			return nil, ErrUnknown
		}
	}
	// Copy object properties to the returned Provider result
	pProps := make([]*pb.Property, len(obj.Properties))
	for x, oProp := range obj.Properties {
		pProps[x] = &pb.Property{
			Key:   oProp.Key,
			Value: oProp.Value,
		}
	}

	return &pb.Provider{
		Partition:    p.Partition,
		ProviderType: p.ProviderType,
		Name:         obj.Name,
		Uuid:         p.Uuid,
		Generation:   p.Generation,
		Properties:   pProps,
		Tags:         obj.Tags,
	}, nil
}

// ProviderList streams zero or more Provider objects back to the client that
// match a set of optional filters
func (s *Server) ProviderList(
	req *pb.ProviderListRequest,
	stream pb.RunmAPI_ProviderListServer,
) error {
	// TODO(jaypipes): Transform the supplied generic filters into the more
	// specific UuidFilter or NameFilter objects accepted by the metadata
	// service
	mfils := make([]*metapb.ObjectFilter, 0)
	mfils = append(mfils, &metapb.ObjectFilter{
		ObjectTypeFilter: &metapb.ObjectTypeFilter{
			CodeFilter: &metapb.CodeFilter{
				Code:      "runm.provider",
				UsePrefix: false,
			},
		},
	})
	// Grab the basic object information from the metadata service first
	objs, err := s.objectsGetMatching(req.Session, mfils)
	if err != nil {
		return err
	}

	if len(objs) == 0 {
		return nil
	}

	// TODO(jaypipes): Create a set of respb.ProviderFilter objects and grab
	// provider-specific information from the runm-resource service

	for _, obj := range objs {
		p := &pb.Provider{
			Partition: obj.Partition,
			Name:      obj.Name,
			Uuid:      obj.Uuid,
		}
		if err = stream.Send(p); err != nil {
			return err
		}
	}
	return nil
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
	obj := &metapb.Object{
		Partition:  input.Partition,
		ObjectType: "runm.provider",
		Uuid:       input.Uuid,
		Name:       input.Name,
		Tags:       input.Tags,
	}
	if input.Properties != nil {
		props := make([]*metapb.Property, 0)
		for key, val := range input.Properties {
			props = append(props, &metapb.Property{
				Key:   key,
				Value: val,
			})
		}
		obj.Properties = props
	}
	createdObj, err := s.objectCreate(req.Session, obj)
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

	// Copy object properties to the returned Provider result
	pProps := make([]*pb.Property, len(obj.Properties))
	for x, oProp := range obj.Properties {
		pProps[x] = &pb.Property{
			Key:   oProp.Key,
			Value: oProp.Value,
		}
	}

	return &pb.ProviderCreateResponse{
		Provider: &pb.Provider{
			Uuid:         input.Uuid,
			Name:         input.Name,
			Partition:    createdObj.Partition,
			ProviderType: input.ProviderType,
			Generation:   resProv.Generation,
			Properties:   pProps,
			Tags:         input.Tags,
		},
	}, nil
}

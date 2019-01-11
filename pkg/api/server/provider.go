package server

import (
	"context"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/runmachine-io/runmachine/pkg/api/types"
	"github.com/runmachine-io/runmachine/pkg/errors"
	metapb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	respb "github.com/runmachine-io/runmachine/pkg/resource/proto"
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
	mfils := make([]*metapb.ObjectFilter, 0)
	// If we get, for example, a filter on a non-existent partition, we
	// increment this variable. If the number of invalid conditions is equal to
	// the number of filters, we return an empty stream and don't bother
	// calling to the resource service.
	invalidConds := 0
	// We keep a cache of partition UUIDs that were normalized during filter
	// expansion/solving with the metadata service so that when we pass filters
	// to the resource service, we have those partition UUIDs handy
	partUuidsReqMap := make(map[int][]string, len(req.Any))
	if len(req.Any) > 0 {
		// Transform the supplied generic filters into the more specific
		// UuidFilter or NameFilter objects accepted by the metadata service
		for x, filter := range req.Any {
			mfil := &metapb.ObjectFilter{
				ObjectTypeFilter: &metapb.ObjectTypeFilter{
					CodeFilter: &metapb.CodeFilter{
						Code:      "runm.provider",
						UsePrefix: false,
					},
				},
			}
			if filter.PrimaryFilter != nil {
				if util.IsUuidLike(filter.PrimaryFilter.Search) {
					mfil.UuidFilter = &metapb.UuidFilter{
						Uuid: filter.PrimaryFilter.Search,
					}
				} else {
					mfil.NameFilter = &metapb.NameFilter{
						Name:      filter.PrimaryFilter.Search,
						UsePrefix: filter.PrimaryFilter.UsePrefix,
					}
				}
			}
			if filter.PartitionFilter != nil {
				// The user may have specified a partition UUID or a partition
				// name with an optional prefix. We "expand" this by asking the
				// metadata service for the partitions matching this
				// name-or-UUID filter and then we pass those partition UUIDs
				// in the object filter.
				partObjs, err := s.partitionsGetMatchingFilter(
					req.Session, filter.PartitionFilter,
				)
				if err != nil {
					return err
				}
				if len(partObjs) == 0 {
					// This filter will never return any objects since the
					// searched-for partition term didn't match any partitions
					invalidConds += 1
					continue
				}
				partUuids := make([]string, len(partObjs))
				for x, partObj := range partObjs {
					partUuids[x] = partObj.Uuid
				}
				mfil.PartitionFilter = &metapb.UuidsFilter{
					Uuids: partUuids,
				}
				// Save in our cache so that the request service filters can
				// use the normalized partition UUIDs
				partUuidsReqMap[x] = partUuids
			}
			mfils = append(mfils, mfil)
		}

	} else {
		// Just get all provider objects from the metadata service
		mfils = append(mfils, &metapb.ObjectFilter{
			ObjectTypeFilter: &metapb.ObjectTypeFilter{
				CodeFilter: &metapb.CodeFilter{
					Code:      "runm.provider",
					UsePrefix: false,
				},
			},
		})

	}

	if len(req.Any) > 0 && len(req.Any) == invalidConds {
		// No point going further, since all filters will return 0 results
		s.log.L3(
			"ProviderList: returning nil since all filters evaluated to " +
				"impossible conditions",
		)
		return nil
	}

	// Grab the basic object information from the metadata service first
	objs, err := s.objectsGetMatching(req.Session, mfils)
	if err != nil {
		return err
	}

	if len(objs) == 0 {
		return nil
	}

	// If the user specified one or more UUIDs or names in the incoming API
	// provider filters, the metadata service will have already handled the
	// translation/lookup to UUIDs, so we can pass the returned object's UUIDs
	// to the resource service's ProviderFilter.UuidFilter below.
	primaryFiltered := false

	objMap := make(map[string]*metapb.Object, len(objs))
	for _, obj := range objs {
		objMap[obj.Uuid] = obj
	}

	var uuids []string
	if primaryFiltered {
		uuids = make([]string, len(objMap))
		x := 0
		for uuid, _ := range objMap {
			uuids[x] = uuid
			x += 1
		}
	}

	// Create a set of respb.ProviderFilter objects and grab provider-specific
	// information from the runm-resource service. For now, we only supply
	// filters to the resource service's ProviderList API call if there were
	// filters passed to the API service's ProviderList API call.
	rfils := make([]*respb.ProviderFilter, 0)
	if len(req.Any) > 0 {
		for x, f := range req.Any {
			rfil := &respb.ProviderFilter{}
			if f.PartitionFilter != nil {
				rfil.PartitionFilter = &respb.UuidFilter{
					Uuids: partUuidsReqMap[x],
				}
			}
			if f.ProviderTypeFilter != nil {
				// TODO(jaypipes): Expand the API SearchFilter for provider
				// types into a []string{} of provider type codes by calling
				// the ProviderTypeList metadata service API. For now, just
				// pass in the Search term as an exact match...
				rfil.ProviderTypeFilter = &respb.CodeFilter{
					Codes: []string{f.ProviderTypeFilter.Search},
				}
			}
			if primaryFiltered {
				rfil.UuidFilter = &respb.UuidFilter{
					Uuids: uuids,
				}
			}
			rfils = append(rfils, rfil)
		}
	}

	// OK, now we grab the provider-specific information from the resource
	// service and mash the generic object information into the returned API
	// Provider structs
	provs, err := s.providersGetMatching(req.Session, rfils)
	if err != nil {
		return err
	}
	for _, prov := range provs {
		obj, exists := objMap[prov.Uuid]
		if !exists {
			s.log.ERR(
				"DATA CORRUPTION! provider with UUID %s returned from "+
					"resource service but no matching object exists in "+
					"metadata service!",
				prov.Uuid,
			)
			continue
		}
		p := &pb.Provider{
			Partition:    obj.Partition,
			Name:         obj.Name,
			Uuid:         obj.Uuid,
			ProviderType: prov.ProviderType,
			Generation:   prov.Generation,
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
	// Make sure we are only passing the partition's UUID, which the created
	// object in the metadata service will have returned.
	input.Partition = createdObj.Partition

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

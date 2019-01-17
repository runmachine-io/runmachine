package server

import (
	"context"
	"io"

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

// ProviderDelete removes one or more providers from backend storage along with
// their associated object metadata in the metadata service.
func (s *Server) ProviderDelete(
	ctx context.Context,
	req *pb.ProviderDeleteRequest,
) (*pb.DeleteResponse, error) {
	if len(req.Any) == 0 {
		return nil, ErrAtLeastOneProviderFilterRequired
	}

	provs, err := s.providersGetMatching(req.Session, req.Any)
	if err != nil {
		return nil, err
	}

	uuids := make([]string, len(provs))
	for x, prov := range provs {
		uuids[x] = prov.Uuid
	}

	// TODO(jaypipes): Archive the provider information?

	// Delete the provider from the resource service
	if err = s.providerDelete(req.Session, uuids); err != nil {
		return nil, err
	}

	// And now delete the provider object from the metadata service
	if err = s.objectDelete(req.Session, uuids); err != nil {
		// TODO(jaypipes): Use Taskflow-oriented library to undo the delete
		// that happened above in the resource service.
		return nil, err
	}

	// TODO(jaypipes): Send an event notification

	return &pb.DeleteResponse{
		NumDeleted: uint64(len(provs)),
	}, nil
}

func isValidSingleProviderFilter(f *pb.ProviderFilter) bool {
	return f != nil && f.PrimaryFilter != nil && f.PrimaryFilter.Search != ""
}

// ProviderGet looks up a provider by UUID or name and returns a Provider
// protobuf message.
func (s *Server) ProviderGet(
	ctx context.Context,
	req *pb.ProviderGetRequest,
) (*pb.Provider, error) {
	if !isValidSingleProviderFilter(req.Filter) {
		return nil, ErrSearchRequired
	}
	var err error
	search := req.Filter.PrimaryFilter.Search
	if !util.IsUuidLike(search) {
		// Look up the provider's UUID in the metadata service by name
		search, err = s.uuidFromName(req.Session, "runm.provider", search)
		if err != nil {
			return nil, err
		}
	}
	p, err := s.providerGetByUuid(req.Session, search)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// providerGetByUuid returns a provider matching the supplied UUID key. If no
// such provider could be found, returns (nil, ErrNotFound)
func (s *Server) providerGetByUuid(
	sess *pb.Session,
	uuid string,
) (*pb.Provider, error) {
	// Grab the provider record from the resource service
	req := &respb.ProviderGetRequest{
		Session: resSession(sess),
		Uuid:    uuid,
	}
	rc, err := s.resClient()
	if err != nil {
		return nil, err
	}
	prec, err := rc.ProviderGet(context.Background(), req)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.NotFound {
				return nil, ErrNotFound
			}
		}
		s.log.ERR(
			"failed to retrieve provider with UUID %s: %s",
			uuid, err,
		)
		return nil, ErrUnknown
	}
	// Grab the object from the metadata service
	obj, err := s.objectFromUuid(sess, uuid)
	if err != nil {
		if err == errors.ErrNotFound {
			s.log.ERR(
				"DATA CORRUPTION! failed getting object with "+
					"UUID %s: object with UUID %s does not exist in metadata "+
					"service but providerGetByUuid returned a provider",
				uuid, err,
			)
			return nil, ErrUnknown
		} else {
			s.log.ERR("failed getting object with UUID %s: %s", uuid, err)
			return nil, ErrUnknown
		}
	}
	return apiProviderFromComponents(prec, obj), nil
}

// mergeProviderWithObject simply takes a resource service Provider and a
// metadata Object and merges the object information into the API Provider's
// generic object fields (like name, tags, properties, etc), returning an API
// provider object from the combined data
func apiProviderFromComponents(
	p *respb.Provider,
	obj *metapb.Object,
) *pb.Provider {
	// Copy object properties to the returned Provider result
	props := make([]*pb.Property, len(obj.Properties))
	for x, oProp := range obj.Properties {
		props[x] = &pb.Property{
			Key:   oProp.Key,
			Value: oProp.Value,
		}
	}

	return &pb.Provider{
		Partition:    p.Partition,
		ProviderType: p.ProviderType,
		Name:         obj.Name,
		Uuid:         obj.Uuid,
		Generation:   p.Generation,
		Properties:   props,
		Tags:         obj.Tags,
	}
}

// ProviderList streams zero or more Provider objects back to the client that
// match a set of optional filters
func (s *Server) ProviderList(
	req *pb.ProviderListRequest,
	stream pb.RunmAPI_ProviderListServer,
) error {
	provs, err := s.providersGetMatching(req.Session, req.Any)
	if err != nil {
		return err
	}
	for _, prov := range provs {
		if err = stream.Send(prov); err != nil {
			return err
		}
	}
	return nil
}

// providersGetMatching returns a slice of pointers to API Provider messages
// matching any of a set of API ProviderFilter messages.
func (s *Server) providersGetMatching(
	sess *pb.Session,
	any []*pb.ProviderFilter,
) ([]*pb.Provider, error) {
	res := make([]*pb.Provider, 0)
	mfils := make([]*metapb.ObjectFilter, 0)
	// If the user specified one or more UUIDs or names in the incoming API
	// provider filters, the metadata service will have already handled the
	// translation/lookup to UUIDs, so we can pass the returned object's UUIDs
	// to the resource service's ProviderFilter.UuidFilter below.
	primaryFiltered := false
	// If we get, for example, a filter on a non-existent partition, we
	// increment this variable. If the number of invalid conditions is equal to
	// the number of filters, we return an empty stream and don't bother
	// calling to the resource service.
	invalidConds := 0
	// We keep a cache of partition UUIDs that were normalized during filter
	// expansion/solving with the metadata service so that when we pass filters
	// to the resource service, we have those partition UUIDs handy
	partUuidsReqMap := make(map[int][]string, len(any))
	if len(any) > 0 {
		// Transform the supplied generic filters into the more specific
		// UuidFilter or NameFilter objects accepted by the metadata service
		for x, filter := range any {
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
				primaryFiltered = true
			}
			if filter.PartitionFilter != nil {
				// The user may have specified a partition UUID or a partition
				// name with an optional prefix. We "expand" this by asking the
				// metadata service for the partitions matching this
				// name-or-UUID filter and then we pass those partition UUIDs
				// in the object filter.
				partObjs, err := s.partitionsGetMatchingFilter(
					sess, filter.PartitionFilter,
				)
				if err != nil {
					return nil, err
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

	if len(any) > 0 && len(any) == invalidConds {
		// No point going further, since all filters will return 0 results
		s.log.L3(
			"ProviderList: returning nil since all filters evaluated to " +
				"impossible conditions",
		)
		return res, nil
	}

	// Grab the basic object information from the metadata service first
	objs, err := s.objectsGetMatching(sess, mfils)
	if err != nil {
		return nil, err
	}

	if len(objs) == 0 {
		return res, nil
	}

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
	if len(any) > 0 {
		for x, f := range any {
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
	rc, err := s.resClient()
	if err != nil {
		return nil, err
	}
	req := &respb.ProviderListRequest{
		Session: resSession(sess),
		Any:     rfils,
	}
	stream, err := rc.ProviderList(context.Background(), req)
	if err != nil {
		return nil, err
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		obj, exists := objMap[msg.Uuid]
		if !exists {
			s.log.ERR(
				"DATA CORRUPTION! provider with UUID %s returned from "+
					"resource service but no matching object exists in "+
					"metadata service!",
				msg.Uuid,
			)
			continue
		}
		p := apiProviderFromComponents(msg, obj)
		res = append(res, p)
	}
	return res, nil
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

// validateProviderDefinitionSetRequest ensures that the data the user sent in
// the request payload can be unmarshal'd properly into YAML and that the data
// is valid
func (s *Server) validateProviderDefinitionSetRequest(
	req *pb.CreateRequest,
) (*pb.ProviderDefinition, error) {
	var input types.ProviderDefinition
	if err := yaml.Unmarshal(req.Payload, &input); err != nil {
		return nil, err
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check that the supplied partition exists
	part, err := s.partitionGet(req.Session, input.Partition)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, errPartitionNotFound(input.Partition)
		}
		s.log.ERR("failed checking provider definition's partition: %s", err)
		return nil, ErrUnknown
	}
	partUuid := part.Uuid

	propPerms := make([]*pb.PropertyPermissions, 0)

	// Ensure that we've got some default access permissions for any properties
	// that have been defined on the provider definition
	for propKey, propDef := range input.PropertyDefinitions {
		if len(propDef.Permissions) == 0 {
			s.log.L3(
				"setting default permissions on provider definition "+
					"in partition '%s' for property key '%s' to READ/WRITE "+
					"for project '%s' and READ any",
				partUuid, propKey, req.Session.Project,
			)
			propPerms = append(propPerms,
				&pb.PropertyPermissions{
					Key: propKey,
					Permissions: []*pb.PropertyPermission{
						&pb.PropertyPermission{
							Project: req.Session.Project,
							Permission: types.PERMISSION_READ |
								types.PERMISSION_WRITE,
						},
						&pb.PropertyPermission{
							Permission: types.PERMISSION_READ,
						},
					},
				},
			)
		} else {
			// Make sure that the project that created the provider definition
			// can read and write the properties defined on it...
			foundProj := false
			for _, perm := range propDef.Permissions {
				if perm.Project != "" && perm.Project == req.Session.Project {
					permCode := perm.PermissionUint32()
					if (permCode & types.PERMISSION_WRITE) == 0 {
						s.log.L1(
							"added missing WRITE permission for "+
								"provider definition in partition '%s' "+
								"for property key '%s' in project '%s'",
							partUuid, propKey, perm.Project,
						)
						permCode |= types.PERMISSION_WRITE
					}
					foundProj = true
					propPerms = append(propPerms,
						&pb.PropertyPermissions{
							Key: propKey,
							Permissions: []*pb.PropertyPermission{
								&pb.PropertyPermission{
									Project:    perm.Project,
									Role:       perm.Role,
									Permission: permCode,
								},
							},
						},
					)
					break
				}
			}
			if !foundProj {
				s.log.L1(
					"added missing WRITE permission for provider definition "+
						"in partition '%s' for property key '%s' in project '%s'",
					partUuid, propKey, req.Session.Project,
				)
				propPerms = append(propPerms,
					&pb.PropertyPermissions{
						Key: propKey,
						Permissions: []*pb.PropertyPermission{
							&pb.PropertyPermission{
								Project: req.Session.Project,
								Permission: types.PERMISSION_READ |
									types.PERMISSION_WRITE,
							},
						},
					},
				)
			}
		}
	}
	return &pb.ProviderDefinition{
		Partition:           partUuid,
		Schema:              input.JSONSchemaString(),
		PropertyPermissions: propPerms,
	}, nil
}

// ProviderDefinitionSet creates or updates the schema and property permissions
// for providers in a particular partition
func (s *Server) ProviderDefinitionSet(
	ctx context.Context,
	req *pb.CreateRequest,
) (*pb.ProviderDefinitionSetResponse, error) {
	// TODO(jaypipes): AUTHZ check if user can write definitions

	def, err := s.validateProviderDefinitionSetRequest(req)
	if err != nil {
		return nil, err
	}

	// copy API property permissions to metadata property permissions
	metaPropPerms := make([]*metapb.PropertyPermissions, len(def.PropertyPermissions))
	for x, apiPropPerms := range def.PropertyPermissions {
		metaPropKeyPerms := make(
			[]*metapb.PropertyPermission, len(apiPropPerms.Permissions),
		)
		for y, apiPropKeyPerm := range apiPropPerms.Permissions {
			metaPropKeyPerms[y] = &metapb.PropertyPermission{
				Project:    apiPropKeyPerm.Project,
				Role:       apiPropKeyPerm.Role,
				Permission: apiPropKeyPerm.Permission,
			}
		}
		metaPropPerms[x] = &metapb.PropertyPermissions{
			Key:         apiPropPerms.Key,
			Permissions: metaPropKeyPerms,
		}
	}

	odef := &metapb.ObjectDefinition{
		Partition:           def.Partition,
		ObjectType:          "runm.provider",
		Schema:              def.Schema,
		PropertyPermissions: metaPropPerms,
	}
	if _, err := s.objectDefinitionSet(req.Session, odef); err != nil {
		s.log.ERR(
			"failed setting object definition for runm.provider objects "+
				"in partition '%s'",
			def.Partition,
		)
		return nil, err
	}

	return &pb.ProviderDefinitionSetResponse{
		ProviderDefinition: def,
	}, nil
}

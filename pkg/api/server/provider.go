package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	yaml "gopkg.in/yaml.v2"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/runmachine-io/runmachine/pkg/api/types"
	"github.com/runmachine-io/runmachine/pkg/errors"
	metapb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	respb "github.com/runmachine-io/runmachine/pkg/resource/proto"
	"github.com/runmachine-io/runmachine/pkg/util"
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
) (*pb.Provider, error) {
	var input types.Provider
	if err := yaml.Unmarshal(req.Payload, &input); err != nil {
		return nil, err
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}

	partUuid := ""
	if input.Partition != "" {
		// Check that the supplied partition exists, and if the user supplied a
		// partition name, translate it to a partition UUID
		part, err := s.partitionGet(req.Session, input.Partition)
		if err != nil {
			return nil, err
		}
		partUuid = part.Uuid
	}

	// Check that the supplied provider type exists
	ptCode := input.ProviderType
	if _, err := s.providerTypeGetByCode(req.Session, ptCode); err != nil {
		return nil, err
	}

	// Grab the provider definition for this partition and use it to validate
	// the supplied provider attributes and properties
	inputJson, err := json.Marshal(&input)
	if err != nil {
		return nil, err
	}
	odef, err := s.providerDefinitionGet(req.Session, partUuid)
	if err != nil {
		return nil, err
	}
	schemaLoader := gojsonschema.NewStringLoader(odef.Schema)
	docLoader := gojsonschema.NewBytesLoader(inputJson)
	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return nil, err
	}
	if !result.Valid() {
		msg := "Error: provider not valid:\n"
		for _, err := range result.Errors() {
			msg += fmt.Sprintf("- %s\n", err)
		}
		return nil, fmt.Errorf(msg)
	}

	props := make([]*pb.Property, 0)
	if input.Properties != nil {
		for key, val := range input.Properties {
			props = append(props, &pb.Property{
				Key:   key,
				Value: propertyValueString(val),
			})
		}
	}

	return &pb.Provider{
		Partition:    partUuid,
		ProviderType: ptCode,
		Name:         input.Name,
		Uuid:         input.Uuid,
		Tags:         input.Tags,
		Properties:   props,
	}, nil
}

func propertyValueString(v interface{}) string {
	switch vt := v.(type) {
	case string:
		return v.(string)
	case int:
		return fmt.Sprintf("%s", v)
	default:
		fmt.Printf("found unknown type for value: %s", vt)
		return ""
	}
}

func (s *Server) ProviderCreate(
	ctx context.Context,
	req *pb.CreateRequest,
) (*pb.ProviderCreateResponse, error) {
	// TODO(jaypipes): AUTHZ check if user can write objects

	p, err := s.validateProviderCreateRequest(req)
	if err != nil {
		return nil, err
	}

	s.log.L3(
		"creating new provider in partition %s with name %s...",
		p.Partition, p.Name,
	)

	// First save the object in the metadata service
	obj := &metapb.Object{
		Partition:  p.Partition,
		ObjectType: "runm.provider",
		Uuid:       p.Uuid,
		Name:       p.Name,
		Tags:       p.Tags,
	}
	if len(p.Properties) > 0 {
		props := make([]*metapb.Property, len(p.Properties))
		for x, prop := range p.Properties {
			props[x] = &metapb.Property{
				Key:   prop.Key,
				Value: prop.Value,
			}
		}
		obj.Properties = props
	}
	if err := s.objectCreate(req.Session, obj); err != nil {
		return nil, err
	}

	// The new object may have set the UUID if it was empty from the user
	p.Uuid = obj.Uuid

	// Next save the provider record in the resource service
	if err := s.providerCreate(req.Session, p); err != nil {
		return nil, err
	}
	s.log.L1(
		"created new provider with UUID %s in partition %s with name %s",
		p.Uuid, p.Partition, p.Name,
	)

	return &pb.ProviderCreateResponse{
		Provider: p,
	}, nil
}

// ProviderDefinitionGet looks up a provider definition by partition UUID or
// name and returns a ProviderDefinition protobuf message.
func (s *Server) ProviderDefinitionGet(
	ctx context.Context,
	req *pb.ProviderDefinitionGetRequest,
) (*pb.ObjectDefinition, error) {
	partUuid := ""
	if req.Partition != "" {
		// Translate any supplied partition identifier into a UUID
		part, err := s.partitionGet(req.Session, req.Partition)
		if err != nil {
			return nil, err
		}
		partUuid = part.Uuid
	}
	odef, err := s.providerDefinitionGet(req.Session, partUuid)
	if err != nil {
		return nil, err
	}

	// copy metadata property permissions to API property permissions
	apiPropPerms := make([]*pb.PropertyPermissions, len(odef.PropertyPermissions))
	for x, metaPropPerms := range odef.PropertyPermissions {
		apiPropKeyPerms := make(
			[]*pb.PropertyPermission, len(metaPropPerms.Permissions),
		)
		for y, metaPropKeyPerm := range metaPropPerms.Permissions {
			apiPropKeyPerms[y] = &pb.PropertyPermission{
				Project:    metaPropKeyPerm.Project,
				Role:       metaPropKeyPerm.Role,
				Permission: metaPropKeyPerm.Permission,
			}
		}
		apiPropPerms[x] = &pb.PropertyPermissions{
			Key:         metaPropPerms.Key,
			Permissions: apiPropKeyPerms,
		}
	}
	return &pb.ObjectDefinition{
		Schema:              odef.Schema,
		PropertyPermissions: apiPropPerms,
	}, nil
}

// validateProviderDefinitionSetRequest ensures that the data the user sent in
// the request payload can be unmarshal'd properly into YAML and that the data
// is valid
func (s *Server) validateProviderDefinitionSetRequest(
	req *pb.ProviderDefinitionSetRequest,
) (*pb.ObjectDefinition, error) {
	var input types.ProviderDefinition
	if err := yaml.Unmarshal(req.Payload, &input); err != nil {
		return nil, err
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}

	partDisplay := "GLOBAL"
	if req.Partition != "" {
		// Check that any supplied partition exists, and if the user supplied a
		// partition name, translate it to a partition UUID
		part, err := s.partitionGet(req.Session, req.Partition)
		if err != nil {
			if err == errors.ErrNotFound {
				return nil, errPartitionNotFound(req.Partition)
			}
			s.log.ERR("failed checking provider definition's partition: %s", err)
			return nil, ErrUnknown
		}
		partDisplay = "partition: '" + part.Uuid + "'"
		req.Partition = part.Uuid
	}

	propPerms := make([]*pb.PropertyPermissions, 0)

	// Ensure that we've got some default access permissions for any properties
	// that have been defined on the provider definition
	for propKey, propDef := range input.PropertyDefinitions {
		if len(propDef.Permissions) == 0 {
			s.log.L3(
				"setting default permissions on provider definition "+
					"in %s for property key '%s' to READ/WRITE "+
					"for project '%s' and READ any",
				partDisplay, propKey, req.Session.Project,
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
								"provider definition in %s "+
								"for property key '%s' in project '%s'",
							partDisplay, propKey, perm.Project,
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
						"in %s for property key '%s' in project '%s'",
					partDisplay, propKey, req.Session.Project,
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
	return &pb.ObjectDefinition{
		Schema:              input.JSONSchemaString(),
		PropertyPermissions: propPerms,
	}, nil
}

// ProviderDefinitionSet creates or updates the schema and property permissions
// for providers in a particular partition
func (s *Server) ProviderDefinitionSet(
	ctx context.Context,
	req *pb.ProviderDefinitionSetRequest,
) (*pb.ObjectDefinitionSetResponse, error) {
	// TODO(jaypipes): AUTHZ check if user can write definitions

	odef, err := s.validateProviderDefinitionSetRequest(req)
	if err != nil {
		return nil, err
	}

	// copy API property permissions to metadata property permissions
	metaPropPerms := make([]*metapb.PropertyPermissions, len(odef.PropertyPermissions))
	for x, apiPropPerms := range odef.PropertyPermissions {
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

	metadef := &metapb.ObjectDefinition{
		Schema:              odef.Schema,
		PropertyPermissions: metaPropPerms,
	}
	if _, err := s.providerDefinitionSet(req.Session, metadef, req.Partition); err != nil {
		s.log.ERR(
			"failed setting object definition for runm.provider objects "+
				"in partition '%s'",
			req.Partition,
		)
		return nil, err
	}

	return &pb.ObjectDefinitionSetResponse{
		ObjectDefinition: odef,
	}, nil
}

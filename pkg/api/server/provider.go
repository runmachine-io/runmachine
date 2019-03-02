package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ghodss/yaml"
	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apipb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/runmachine-io/runmachine/pkg/api/types"
	"github.com/runmachine-io/runmachine/pkg/errors"
	"github.com/runmachine-io/runmachine/pkg/util"
	pb "github.com/runmachine-io/runmachine/proto"
)

// ProviderDelete removes one or more providers from backend storage along with
// their associated object metadata in the metadata service.
func (s *Server) ProviderDelete(
	ctx context.Context,
	req *apipb.ProviderDeleteRequest,
) (*apipb.DeleteResponse, error) {
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
	if err = s.providerDeleteByUuids(req.Session, uuids); err != nil {
		return nil, err
	}

	// And now delete the provider object from the metadata service
	if err = s.objectDelete(req.Session, uuids); err != nil {
		// TODO(jaypipes): Use Taskflow-oriented library to undo the delete
		// that happened above in the resource service.
		return nil, err
	}

	// TODO(jaypipes): Send an event notification

	return &apipb.DeleteResponse{
		NumDeleted: uint64(len(provs)),
	}, nil
}

// providerDeleteByUuids deletes the provider records from the resource service
// having any of the supplied UUIDs
func (s *Server) providerDeleteByUuids(
	sess *apipb.Session,
	uuids []string,
) error {
	req := &pb.ProviderDeleteByUuidsRequest{
		Session: resSession(sess),
		Uuids:   uuids,
	}
	rc, err := s.resClient()
	_, err = rc.ProviderDeleteByUuids(context.Background(), req)
	if err != nil {
		s.log.ERR(
			"failed deleting providers with UUIDs (%s) in resource service: %s",
			uuids, err,
		)
		return err
	}
	return nil
}

func isValidSingleProviderFilter(f *apipb.ProviderFilter) bool {
	return f != nil && f.PrimaryFilter != nil && f.PrimaryFilter.Search != ""
}

// ProviderGet looks up a provider by UUID or name and returns a Provider
// protobuf message.
func (s *Server) ProviderGet(
	ctx context.Context,
	req *apipb.ProviderGetRequest,
) (*apipb.Provider, error) {
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
	sess *apipb.Session,
	uuid string,
) (*apipb.Provider, error) {
	// Grab the provider record from the resource service
	req := &pb.ProviderGetByUuidRequest{
		Session: resSession(sess),
		Uuid:    uuid,
	}
	rc, err := s.resClient()
	if err != nil {
		return nil, err
	}
	prec, err := rc.ProviderGetByUuid(context.Background(), req)
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

// apiProviderFromComponents takes a resource service Provider and a metadata
// Object and merges the object information into the API Provider's generic
// object fields (like name, tags, properties, etc), returning an API provider
// object from the combined data
func apiProviderFromComponents(
	p *pb.Provider,
	obj *pb.Object,
) *apipb.Provider {
	// Copy object properties to the returned Provider result
	props := make([]*apipb.Property, len(obj.Properties))
	for x, oProp := range obj.Properties {
		props[x] = &apipb.Property{
			Key:   oProp.Key,
			Value: oProp.Value,
		}
	}

	return &apipb.Provider{
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
	req *apipb.ProviderListRequest,
	stream apipb.RunmAPI_ProviderListServer,
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
	sess *apipb.Session,
	any []*apipb.ProviderFilter,
) ([]*apipb.Provider, error) {
	res := make([]*apipb.Provider, 0)
	mfils := make([]*pb.ObjectFilter, 0)
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
			mfil := &pb.ObjectFilter{
				ObjectTypeFilter: &pb.ObjectTypeFilter{
					CodeFilter: &pb.CodeFilter{
						Code:      "runm.provider",
						UsePrefix: false,
					},
				},
			}
			if filter.PrimaryFilter != nil {
				if util.IsUuidLike(filter.PrimaryFilter.Search) {
					mfil.UuidFilter = &pb.UuidFilter{
						Uuid: filter.PrimaryFilter.Search,
					}
				} else {
					mfil.NameFilter = &pb.NameFilter{
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
				mfil.PartitionFilter = &pb.UuidsFilter{
					Uuids: partUuids,
				}
				// Save in our cache so that the request service filters can
				// use the normalized partition UUIDs
				partUuidsReqMap[x] = partUuids
			}
			if filter.PropertyFilter != nil {
				propFilter := filter.PropertyFilter
				// TODO(jaypipes): Once Issue #111 is done, this copying won't
				// be necessary
				metaPropFilter := &pb.PropertyFilter{
					RequireKeys: propFilter.RequireKeys,
					AnyKeys:     propFilter.AnyKeys,
					ForbidKeys:  propFilter.ForbidKeys,
				}
				if propFilter.RequireItems != nil {
					numItems := len(propFilter.RequireItems)
					items := make([]*pb.Property, numItems)
					for x, prop := range propFilter.RequireItems {
						items[x] = &pb.Property{
							Key:   prop.Key,
							Value: prop.Value,
						}
					}
					metaPropFilter.RequireItems = items
				}
				if propFilter.AnyItems != nil {
					numItems := len(propFilter.AnyItems)
					items := make([]*pb.Property, numItems)
					for x, prop := range propFilter.AnyItems {
						items[x] = &pb.Property{
							Key:   prop.Key,
							Value: prop.Value,
						}
					}
					metaPropFilter.AnyItems = items
				}
				if propFilter.ForbidItems != nil {
					numItems := len(propFilter.ForbidItems)
					items := make([]*pb.Property, numItems)
					for x, prop := range propFilter.ForbidItems {
						items[x] = &pb.Property{
							Key:   prop.Key,
							Value: prop.Value,
						}
					}
					metaPropFilter.ForbidItems = items
				}
				mfil.PropertyFilter = metaPropFilter
				primaryFiltered = true
			}
			mfils = append(mfils, mfil)
		}
	} else {
		// Just get all provider objects from the metadata service
		mfils = append(mfils, &pb.ObjectFilter{
			ObjectTypeFilter: &pb.ObjectTypeFilter{
				CodeFilter: &pb.CodeFilter{
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

	objMap := make(map[string]*pb.Object, len(objs))
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

	// Create a set of pb.ProviderFilter objects and grab provider-specific
	// information from the runm-resource service. For now, we only supply
	// filters to the resource service's ProviderList API call if there were
	// filters passed to the API service's ProviderList API call.
	rfils := make([]*pb.ProviderFilter, 0)
	if len(any) > 0 {
		for x, f := range any {
			rfil := &pb.ProviderFilter{}
			if f.PartitionFilter != nil {
				rfil.PartitionFilter = &pb.UuidsFilter{
					Uuids: partUuidsReqMap[x],
				}
			}
			if f.ProviderTypeFilter != nil {
				// TODO(jaypipes): Expand the API SearchFilter for provider
				// types into a []string{} of provider type codes by calling
				// the ProviderTypeList metadata service API. For now, just
				// pass in the Search term as an exact match...
				rfil.ProviderTypeFilter = &pb.CodesFilter{
					Codes: []string{f.ProviderTypeFilter.Search},
				}
			}
			if primaryFiltered {
				rfil.UuidFilter = &pb.UuidsFilter{
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
	req := &pb.ProviderListRequest{
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
	req *apipb.CreateRequest,
) (*apipb.Provider, error) {
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
	odef, err := s.providerDefinitionGetMostExplicit(
		req.Session, partUuid, ptCode,
	)
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

	props := make([]*apipb.Property, 0)
	if input.Properties != nil {
		for key, val := range input.Properties {
			props = append(props, &apipb.Property{
				Key:   key,
				Value: propertyValueString(val),
			})
		}
	}

	return &apipb.Provider{
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
	case int64:
		return fmt.Sprintf("%d", v.(int64))
	case float64:
		// JSON unmarshaling apparently returns all numbers (including
		// integers) as float64. So, I'm not entirely sure how to preserve
		// actual floats (JSON number type)
		return fmt.Sprintf("%d", int(v.(float64)))
	default:
		fmt.Printf("found unknown type for value: %s", vt)
		return ""
	}
}

func (s *Server) ProviderCreate(
	ctx context.Context,
	req *apipb.CreateRequest,
) (*apipb.ProviderCreateResponse, error) {
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
	obj := &pb.Object{
		Partition:  p.Partition,
		ObjectType: "runm.provider",
		Uuid:       p.Uuid,
		Name:       p.Name,
		Tags:       p.Tags,
	}
	if len(p.Properties) > 0 {
		props := make([]*pb.Property, len(p.Properties))
		for x, prop := range p.Properties {
			props[x] = &pb.Property{
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

	return &apipb.ProviderCreateResponse{
		Provider: p,
	}, nil
}

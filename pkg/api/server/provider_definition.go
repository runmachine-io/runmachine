package server

import (
	"context"
	"fmt"

	"github.com/ghodss/yaml"

	"github.com/runmachine-io/runmachine/pkg/api/types"
	"github.com/runmachine-io/runmachine/pkg/errors"
	pb "github.com/runmachine-io/runmachine/proto"
)

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
	ptCode := ""
	if req.ProviderType != "" {
		_, err := s.providerTypeGetByCode(req.Session, req.ProviderType)
		if err != nil {
			return nil, err
		}
		ptCode = req.ProviderType
	}

	var err error
	var odef *pb.ObjectDefinition
	if partUuid != "" {
		if ptCode != "" {
			odef, err = s.providerDefinitionGetByPartitionAndType(
				req.Session, partUuid, ptCode,
			)
			if err != nil {
				return nil, err
			}
		} else {
			odef, err = s.providerDefinitionGetByPartition(
				req.Session, partUuid,
			)
			if err != nil {
				return nil, err
			}
		}
	} else {
		if ptCode != "" {
			odef, err = s.providerDefinitionGetByType(
				req.Session, ptCode,
			)
			if err != nil {
				return nil, err
			}
		} else {
			odef, err = s.providerDefinitionGetGlobalDefault(req.Session)
			if err != nil {
				return nil, err
			}
		}
	}

	// copy metadata property permissions to API property permissions
	// TODO(jaypipes): This will not be necessary when Issue #111 is done and
	// we have a single protobuffer namespace
	apiPropPerms := make(
		[]*pb.PropertyPermissions,
		len(odef.PropertyPermissions),
	)
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

	if req.ProviderType != "" {
		_, err := s.providerTypeGetByCode(req.Session, req.ProviderType)
		if err != nil {
			if err == errors.ErrNotFound {
				return nil, errProviderTypeNotFound(req.ProviderType)
			}
			s.log.ERR(
				"failed checking provider definition's provider type: %s",
				err,
			)
			return nil, ErrUnknown
		}
		partDisplay += " provider type: '" + req.ProviderType + "'"
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
	metaPropPerms := make(
		[]*pb.PropertyPermissions,
		len(odef.PropertyPermissions),
	)
	for x, apiPropPerms := range odef.PropertyPermissions {
		metaPropKeyPerms := make(
			[]*pb.PropertyPermission, len(apiPropPerms.Permissions),
		)
		for y, apiPropKeyPerm := range apiPropPerms.Permissions {
			metaPropKeyPerms[y] = &pb.PropertyPermission{
				Project:    apiPropKeyPerm.Project,
				Role:       apiPropKeyPerm.Role,
				Permission: apiPropKeyPerm.Permission,
			}
		}
		metaPropPerms[x] = &pb.PropertyPermissions{
			Key:         apiPropPerms.Key,
			Permissions: metaPropKeyPerms,
		}
	}

	metadef := &pb.ObjectDefinition{
		Schema:              odef.Schema,
		PropertyPermissions: metaPropPerms,
	}
	_, err = s.providerDefinitionSet(
		req.Session, metadef, req.Partition, req.ProviderType,
	)
	if err != nil {
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

// providerDefinitionGetGlobalDefault returns the global default object
// definition for providers.
//
// If no such object definition could be found, returns (nil, ErrNotFound)
func (s *Server) providerDefinitionGetGlobalDefault(
	sess *pb.Session,
) (*pb.ObjectDefinition, error) {
	req := &pb.ProviderDefinitionGetGlobalDefaultRequest{
		Session: sess,
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	def, err := mc.ProviderDefinitionGetGlobalDefault(
		context.Background(), req,
	)
	if err != nil {
		return nil, err
	}
	return def, nil
}

// providerDefinitionGetByPartition returns the object definition for providers
// that has been set as the override for a supplied partition.
//
// If no such object definition could be found, returns (nil, ErrNotFound)
func (s *Server) providerDefinitionGetByPartition(
	sess *pb.Session,
	partUuid string,
) (*pb.ObjectDefinition, error) {
	req := &pb.ProviderDefinitionGetByPartitionRequest{
		Session:       sess,
		PartitionUuid: partUuid,
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	def, err := mc.ProviderDefinitionGetByPartition(
		context.Background(), req,
	)
	if err != nil {
		return nil, err
	}
	return def, nil
}

// providerDefinitionGetByType returns the object definition for providers that
// has been overridden for the supplied provider type.
//
// If no such object definition could be found, returns (nil, ErrNotFound)
func (s *Server) providerDefinitionGetByType(
	sess *pb.Session,
	provTypeCode string,
) (*pb.ObjectDefinition, error) {
	req := &pb.ProviderDefinitionGetByTypeRequest{
		Session:          sess,
		ProviderTypeCode: provTypeCode,
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	def, err := mc.ProviderDefinitionGetByType(
		context.Background(), req,
	)
	if err != nil {
		return nil, err
	}
	return def, nil
}

// providerDefinitionGetByPartitionAndType returns the object definition for
// providers that has been overridden for the supplied partition and provider
// type.
//
// If no such object definition could be found, returns (nil, ErrNotFound)
func (s *Server) providerDefinitionGetByPartitionAndType(
	sess *pb.Session,
	partUuid string,
	provTypeCode string,
) (*pb.ObjectDefinition, error) {
	req := &pb.ProviderDefinitionGetByPartitionAndTypeRequest{
		Session:          sess,
		PartitionUuid:    partUuid,
		ProviderTypeCode: provTypeCode,
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	def, err := mc.ProviderDefinitionGetByPartitionAndType(
		context.Background(), req,
	)
	if err != nil {
		return nil, err
	}
	return def, nil
}

// providerDefinitionGetMostExplicit returns the object definition that would
// be applied for the supplied partition and provider type.
//
// If a provider definition override has been set for the partition and
// provider type, that object definition will be returned, otherwise...
//
// If a provider definition override has been set for the partition but not the
// provider type, that object definition will be returned, otherwise...
//
// If a provider definition override has been set for the provider type but not
// the partition, that object definition will be returned, otherwise...
//
// If no overrides for partition or provider type have been set, this will return the global default provider definition.
func (s *Server) providerDefinitionGetMostExplicit(
	sess *pb.Session,
	partUuid string,
	provTypeCode string,
) (*pb.ObjectDefinition, error) {
	if partUuid == "" {
		return nil, fmt.Errorf("partUuid parameter must not be empty")
	}
	if provTypeCode == "" {
		return nil, fmt.Errorf("provTypeCode parameter must not be empty")
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}

	// OK, first look to see if there's an override for the partition +
	// provider type
	pptreq := &pb.ProviderDefinitionGetByPartitionAndTypeRequest{
		Session:          sess,
		PartitionUuid:    partUuid,
		ProviderTypeCode: provTypeCode,
	}
	def, err := mc.ProviderDefinitionGetByPartitionAndType(
		context.Background(), pptreq,
	)
	if err != nil {
		if err != errors.ErrNotFound {
			return nil, err
		}
	} else {
		return def, nil
	}

	// We fell through here if there was no partition + provider type override.
	// Next check to see if there's a partition (with no provider type)
	// override.
	preq := &pb.ProviderDefinitionGetByPartitionRequest{
		Session:       sess,
		PartitionUuid: partUuid,
	}
	def, err = mc.ProviderDefinitionGetByPartition(context.Background(), preq)
	if err != nil {
		if err != errors.ErrNotFound {
			return nil, err
		}
	} else {
		return def, nil
	}

	// We fell through here if there was no partition + provider type override
	// and no partition-only override. Next check to see if there's a provider
	// type (no partition) override.
	ptreq := &pb.ProviderDefinitionGetByTypeRequest{
		Session:          sess,
		ProviderTypeCode: provTypeCode,
	}
	def, err = mc.ProviderDefinitionGetByType(context.Background(), ptreq)
	if err != nil {
		if err != errors.ErrNotFound {
			return nil, err
		}
	} else {
		return def, nil
	}

	// Nothing found... fall back on the global default provider definition
	return s.providerDefinitionGetGlobalDefault(sess)
}

// providerDefinitionSet takes an object definition and saves it in the metadata
// service, returning the saved object definition
func (s *Server) providerDefinitionSet(
	sess *pb.Session,
	def *pb.ObjectDefinition,
	partUuid string,
	provTypeCode string,
) (*pb.ObjectDefinition, error) {
	req := &pb.ProviderObjectDefinitionSetRequest{
		Session:          sess,
		ObjectDefinition: def,
		PartitionUuid:    partUuid,
		ProviderTypeCode: provTypeCode,
	}
	mc, err := s.metaClient()
	if err != nil {
		return nil, err
	}
	resp, err := mc.ProviderDefinitionSet(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.ObjectDefinition, nil
}

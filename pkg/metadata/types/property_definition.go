package types

import (
	"fmt"
	"os"
	"strconv"

	apitypes "github.com/runmachine-io/runmachine/pkg/api/types"
	pb "github.com/runmachine-io/runmachine/proto"
)

type PropertyDefinitionMatcher interface {
	Matches(obj *pb.PropertyDefinition) bool
}

// PropertyDefinitionCondition is a class used in filtering property definitions.
// Optional partition and object type PKs have already been expanded from
// user-supplied partition and type filter strings
type PropertyDefinitionCondition struct {
	Partition   *PartitionCondition
	ObjectType  *ObjectTypeCondition
	Uuid        *UuidCondition
	PropertyKey *PropertyKeyCondition
}

func (f *PropertyDefinitionCondition) Matches(obj *pb.PropertyDefinition) bool {
	if !f.Uuid.Matches(obj) {
		return false
	}
	if !f.Partition.Matches(obj) {
		return false
	}
	if !f.ObjectType.Matches(obj) {
		return false
	}
	if !f.PropertyKey.Matches(obj) {
		return false
	}
	return true
}

func (f *PropertyDefinitionCondition) IsEmpty() bool {
	return f.Partition == nil && f.ObjectType == nil && f.PropertyKey == nil && f.Uuid == nil
}

func (f *PropertyDefinitionCondition) String() string {
	attrMap := make(map[string]string, 0)
	if f.Partition != nil {
		attrMap["partition"] = f.Partition.Partition.Uuid
	}
	if f.ObjectType != nil {
		attrMap["object_type"] = f.ObjectType.ObjectType.Code
	}
	if f.Uuid != nil {
		attrMap["uuid"] = f.Uuid.Uuid
	}
	if f.PropertyKey != nil {
		attrMap["key"] = f.PropertyKey.PropertyKey
		attrMap["use_prefix"] = strconv.FormatBool(
			f.PropertyKey.Op == OP_GREATER_THAN_EQUAL,
		)
	}
	attrs := ""
	x := 0
	for k, v := range attrMap {
		if x > 0 {
			attrs += ","
		}
		attrs += k + "=" + v
	}
	return fmt.Sprintf("PropertyDefinitionCondition(%s)", attrs)
}

// PropertyDefinitionWithReferences is a concrete struct containing pointers to
// already-constructed and validated Partition and ObjectType messages. This is
// the struct that is passed to backend storage when creating new property
// schemas, not the protobuffer PropertyDefinition message or the
// api/types/PropertyDefinition struct, neither of which are guaranteed to be
// pre-validated and their relations already expanded.
type PropertyDefinitionWithReferences struct {
	Partition  *pb.Partition
	ObjectType *pb.ObjectType
	Definition *pb.PropertyDefinition
}

// APItoPBPropertySchema onverts an apitypes PropertySchema to the protobuffer
// PropertySchema message that will eb stored in backend storage
func APItoPBPropertySchema(as *apitypes.PropertySchema) *pb.PropertySchema {
	res := &pb.PropertySchema{
		Types:   []pb.PropertySchema_Type{},
		Pattern: as.Pattern,
	}
	if len(as.Types) > 0 {
		for _, astype := range as.Types {
			switch astype {
			case "string":
				res.Types = append(res.Types, pb.PropertySchema_TYPE_STRING)
			case "integer":
				res.Types = append(res.Types, pb.PropertySchema_TYPE_INTEGER)
			case "number":
				res.Types = append(res.Types, pb.PropertySchema_TYPE_NUMBER)
			case "boolean":
				res.Types = append(res.Types, pb.PropertySchema_TYPE_BOOLEAN)
			default:
				fmt.Fprintf(
					os.Stderr,
					"WARNING: unexpected apitypes PropertySchema type: %s",
					astype,
				)
			}
		}
	}
	if as.MultipleOf != nil {
		res.MultipleOf = &pb.UInt64Value{
			Value: uint64(*as.MultipleOf),
		}
	}
	if as.Minimum != nil {
		res.Minimum = &pb.Int64Value{
			Value: int64(*as.Minimum),
		}
	}
	if as.Maximum != nil {
		res.Maximum = &pb.Int64Value{
			Value: int64(*as.Maximum),
		}
	}
	if as.MinLength != nil {
		res.MinimumLength = &pb.UInt64Value{
			Value: uint64(*as.MinLength),
		}
	}
	if as.MaxLength != nil {
		res.MaximumLength = &pb.UInt64Value{
			Value: uint64(*as.MaxLength),
		}
	}
	if as.Format != "" {
		switch as.Format {
		case "date-time":
			res.Format = pb.PropertySchema_FORMAT_DATETIME
		case "date":
			res.Format = pb.PropertySchema_FORMAT_DATE
		case "time":
			res.Format = pb.PropertySchema_FORMAT_TIME
		case "email":
			res.Format = pb.PropertySchema_FORMAT_EMAIL
		case "idn-email":
			res.Format = pb.PropertySchema_FORMAT_IDN_EMAIL
		case "hostname":
			res.Format = pb.PropertySchema_FORMAT_HOSTNAME
		case "idn-hostname":
			res.Format = pb.PropertySchema_FORMAT_IDN_HOSTNAME
		case "ipv4":
			res.Format = pb.PropertySchema_FORMAT_IPV4
		case "ipv6":
			res.Format = pb.PropertySchema_FORMAT_IPV6
		case "uri":
			res.Format = pb.PropertySchema_FORMAT_URI
		case "uri-reference":
			res.Format = pb.PropertySchema_FORMAT_URI_REFERENCE
		case "iri":
			res.Format = pb.PropertySchema_FORMAT_IRI
		case "iri-reference":
			res.Format = pb.PropertySchema_FORMAT_IRI_REFERENCE
		case "uri-template":
			res.Format = pb.PropertySchema_FORMAT_URI_TEMPLATE
		default:
			fmt.Fprintf(
				os.Stderr,
				"WARNING: unexpected apitypes PropertySchema format: %s",
				as.Format,
			)
		}
	}
	return res
}

// APItoPBPropertyPermissions converts the apitypes.PropertyPermissions to
// protobuffer PropertyPermissions that get stored in backend storage
func APItoPBPropertyPermissions(
	apiperms []*apitypes.PropertyPermission,
) []*pb.PropertyPermission {
	res := make([]*pb.PropertyPermission, len(apiperms))
	for x, apiperm := range apiperms {
		// Convert the string "r", "rw" representation to the integer
		// permission code used in backend protobuffer storage
		iperm := apitypes.PERMISSION_NONE
		switch apiperm.Permission {
		case "r":
			iperm = apitypes.PERMISSION_READ
		case "rw":
			iperm = apitypes.PERMISSION_READ | apitypes.PERMISSION_WRITE
		case "w":
			iperm = apitypes.PERMISSION_WRITE
		default:
			iperm = apitypes.PERMISSION_NONE
		}
		pbperm := &pb.PropertyPermission{
			Permission: iperm,
		}
		if apiperm.Project != "" {
			pbperm.Project = &pb.StringValue{
				Value: apiperm.Project,
			}
		}
		if apiperm.Role != "" {
			pbperm.Role = &pb.StringValue{
				Value: apiperm.Role,
			}
		}
		res[x] = pbperm
	}
	return res
}

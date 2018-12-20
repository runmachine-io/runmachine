package commands

import (
	"fmt"
	"strconv"

	apitypes "github.com/runmachine-io/runmachine/pkg/api/types"
	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var (
	propSchemaFormatMap = map[pb.PropertySchema_Format]string{
		pb.PropertySchema_FORMAT_DATETIME:      "date-time",
		pb.PropertySchema_FORMAT_DATE:          "date",
		pb.PropertySchema_FORMAT_TIME:          "time",
		pb.PropertySchema_FORMAT_EMAIL:         "email",
		pb.PropertySchema_FORMAT_IDN_EMAIL:     "idn-email",
		pb.PropertySchema_FORMAT_HOSTNAME:      "hostname",
		pb.PropertySchema_FORMAT_IDN_HOSTNAME:  "idn-hostname",
		pb.PropertySchema_FORMAT_IPV4:          "ipv4",
		pb.PropertySchema_FORMAT_IPV6:          "ipv6",
		pb.PropertySchema_FORMAT_URI:           "uri",
		pb.PropertySchema_FORMAT_URI_REFERENCE: "uri-reference",
		pb.PropertySchema_FORMAT_IRI:           "iri",
		pb.PropertySchema_FORMAT_IRI_REFERENCE: "iri-reference",
		pb.PropertySchema_FORMAT_URI_TEMPLATE:  "uri-template",
	}
	propSchemaTypeMap = map[pb.PropertySchema_Type]string{
		pb.PropertySchema_TYPE_STRING:  "string",
		pb.PropertySchema_TYPE_INTEGER: "integer",
		pb.PropertySchema_TYPE_NUMBER:  "number",
		pb.PropertySchema_TYPE_BOOLEAN: "boolean",
	}
)

var propertyDefinitionCommand = &cobra.Command{
	Use:   "property-definition",
	Short: "Manipulate property definition information",
}

func init() {
	propertyDefinitionCommand.AddCommand(propertyDefinitionListCommand)
	propertyDefinitionCommand.AddCommand(propertyDefinitionGetCommand)
	propertyDefinitionCommand.AddCommand(propertyDefinitionSetCommand)
	propertyDefinitionCommand.AddCommand(propertyDefinitionDeleteCommand)
}

func printPropertySchema(obj *pb.PropertySchema) {
	fmt.Printf("Schema:\n")
	fmt.Printf("  Types:\n")
	for _, t := range obj.Types {
		fmt.Printf("    - %s\n", propSchemaTypeMap[t])
	}
	if obj.MultipleOf != nil {
		fmt.Printf("  Multiple of: %d\n", obj.MultipleOf.Value)
	}
	if obj.Minimum != nil {
		fmt.Printf("  Minimum: %d\n", obj.Minimum.Value)
	}
	if obj.Maximum != nil {
		fmt.Printf("  Maximum: %d\n", obj.Maximum.Value)
	}
	if obj.MinimumLength != nil {
		fmt.Printf("  Minimum length: %d\n", obj.MinimumLength.Value)
	}
	if obj.MaximumLength != nil {
		fmt.Printf("  Maximum length: %d\n", obj.MaximumLength.Value)
	}
	if obj.Format != pb.PropertySchema_FORMAT_NONE {
		fmt.Printf("  Format: %s\n", propSchemaFormatMap[obj.Format])
	}
	if obj.Pattern != "" {
		fmt.Printf("  Pattern: %s", obj.Pattern)
	}
}

func printPropertyDefinition(obj *pb.PropertyDefinition) {
	fmt.Printf("Partition:    %s\n", obj.Partition)
	fmt.Printf("Type:         %s\n", obj.Type)
	fmt.Printf("Key:          %s\n", obj.Key)
	fmt.Printf("Required:     %s\n", strconv.FormatBool(obj.IsRequired))
	if len(obj.Permissions) > 0 {
		fmt.Printf("Permissions:\n")
		for x, perm := range obj.Permissions {
			permStr := ""
			if perm.Project != nil {
				permStr += "project: " + perm.Project.Value
			}
			if perm.Role != nil {
				if len(permStr) > 0 {
					permStr += " "
				}
				permStr += "role: " + perm.Role.Value
			}
			if len(permStr) > 0 {
				permStr += " "
			}
			permStr += "permission: "
			readBit := perm.Permission & apitypes.PERMISSION_READ
			writeBit := perm.Permission & apitypes.PERMISSION_WRITE
			if readBit != 0 {
				if writeBit != 0 {
					permStr += "READ/WRITE"
				} else {
					permStr += "READ"
				}
			} else if writeBit != 0 {
				permStr += "WRITE"
			} else {
				permStr += "NONE (Deny)"
			}
			fmt.Printf("  %d: %s\n", x+1, permStr)
		}
	}
	if obj.Schema != nil {
		printPropertySchema(obj.Schema)
	}
}

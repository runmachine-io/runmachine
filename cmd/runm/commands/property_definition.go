package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	apitypes "github.com/runmachine-io/runmachine/pkg/api/types"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

const (
	usagePropertyDefinitionFilterOption = `optional filter to apply.

--filter <filter expression>

Multiple filters may be applied to the property-schema list operation. Each
filter's field expression is evaluated using an "AND" condition. Multiple
filters are evaluated using an "OR" condition.

The <filter expression> value is a whitespace-separated set of $field=$value
expressions to filter by. $field may be any of the following:

- partition: UUID or name of the partition the property definition belongs to
- type: code of the object type (:see runm object-type list)
- key: the property key to list property definitions for
- uuid: the UUID of the property definition itself

The $value should be an identifier or name for the $field. You can use an
asterisk (*) to indicate a prefix match. For example, to list all property
schemas for objects of type "runm.machine" for property keys that start with
the string "arch", you would use --filter "type=runm.machine key=arch*"

Examples:

Find all property definitions that apply to runm.image objects in a partition
beginning with "east":

--filter "type=runm.image partition=east*"

Find all property definitions that apply to runm.machine objects OR runm.image
objects that are a partition called part0:

--filter "type=runm.machine partition=part0" \
--filter "type=runm.image partition=part0"

Find a property definition with a UUID of "f287341160ee4feba4012eb7f8125b82":

--filter "uuid=f287341160ee4feba4012eb7f8125b82"
`
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

func buildPropertyDefinitionFilters() []*pb.PropertyDefinitionFilter {
	filters := make([]*pb.PropertyDefinitionFilter, 0)
	// Each --filter <field expression> supplied by the user will have one or
	// more $field=$value segments to it, separated by spaces. Split those
	// $field=$value pairs up and evaluate each $field and $value string for
	// fitness
	for _, f := range cliFilters {
		fieldExprs := strings.Fields(f)
		filter := &pb.PropertyDefinitionFilter{}
		for _, fieldExpr := range fieldExprs {
			kvs := strings.SplitN(fieldExpr, "=", 2)
			if len(kvs) != 2 {
				fmt.Fprintf(os.Stderr, errMsgFieldExprFormat, fieldExpr)
				os.Exit(1)
			}
			field := kvs[0]
			value := kvs[1]
			usePrefix := false
			if strings.HasSuffix(value, "*") {
				usePrefix = true
				value = strings.TrimRight(value, "*")
			}
			switch field {
			case "partition":
				filter.Partition = &pb.PartitionFilter{
					Search:    value,
					UsePrefix: usePrefix,
				}
			case "type":
				filter.ObjectType = &pb.ObjectTypeFilter{
					Search:    value,
					UsePrefix: usePrefix,
				}
			case "uuid":
				filter.Uuid = value
			case "key":
				filter.Key = value
				filter.UsePrefix = usePrefix
			default:
				fmt.Fprintf(
					os.Stderr,
					errMsgUnknownFieldInFieldExpr,
					fieldExpr,
					field,
				)
				os.Exit(1)
			}
		}
		filters = append(filters, filter)
	}
	return filters
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

func printPropertyPermission(obj *pb.PropertyPermission) {
	if obj.Project == nil && obj.Role == nil {
		fmt.Printf("GLOBAL ")
	} else {
		if obj.Project != nil {
			fmt.Printf("PROJECT(" + obj.Project.Value + ") ")
		}
		if obj.Role != nil {
			fmt.Printf("ROLE(" + obj.Role.Value + ") ")
		}
	}
	readBit := obj.Permission & apitypes.PERMISSION_READ
	writeBit := obj.Permission & apitypes.PERMISSION_WRITE
	if readBit != 0 {
		if writeBit != 0 {
			fmt.Printf("READ/WRITE\n")
		} else {
			fmt.Printf("READ\n")
		}
	} else if writeBit != 0 {
		fmt.Printf("WRITE\n")
	} else {
		fmt.Printf("NONE (Deny)\n")
	}
}

func printPropertyDefinition(obj *pb.PropertyDefinition) {
	fmt.Printf("Partition:    %s\n", obj.Partition)
	fmt.Printf("Object Type:  %s\n", obj.ObjectType)
	fmt.Printf("Key:          %s\n", obj.Key)
	fmt.Printf("UUID:         %s\n", obj.Uuid)
	fmt.Printf("Required:     %s\n", strconv.FormatBool(obj.IsRequired))
	if len(obj.Permissions) > 0 {
		fmt.Printf("Permissions:\n")
		for _, perm := range obj.Permissions {
			fmt.Printf("  - ")
			printPropertyPermission(perm)
		}
	}
	if obj.Schema != nil {
		printPropertySchema(obj.Schema)
	}
}

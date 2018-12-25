package commands

import (
	"fmt"
	"os"
	"strings"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

const (
	usageObjectFilterOption = `optional filter to apply.

--filter <filter expression>

Multiple filters may be applied to the object list operation. Each filter's
field expression is evaluated using an "AND" condition. Multiple filters are
evaluated using an "OR" condition.

The <filter expression> value is a whitespace-separated set of $field=$value
expressions to filter by. $field may be any of the following:

- partition: UUID or name of the partition the object belongs to
- type: code of the object type (:see runm object-type list)
- project: identifier of the project the object belongs to
- uuid: the UUID of the object itself
- name: name of the object

The $value should be an identifier or name for the $field. You can use an
asterisk (*) to indicate a prefix match. For example, to list all objects of
type "runm.machine" with names that start with the string "east", you would use
--filter "type=runm.machine object=east*"

Examples:

Find all runm.image objects starting with "db" in the "admin" project:

--filter "type=runm.image object=db* project=admin"

Find all runm.machine objects OR runm.image objects that are a partition called
part0:

--filter "type=runm.machine partition=part0" \
--filter "type=runm.image partition=part0"

Find any object in the "admin" project that has a name starting with "db":

--filter "project=admin name=db*"

Find an object with a UUID of "f287341160ee4feba4012eb7f8125b82":

--filter "uuid=f287341160ee4feba4012eb7f8125b82"
`
	errMsgFieldExprFormat = `ERROR: field expression %s expected to be in the form $field=$value
`
	errMsgUnknownFieldInFieldExpr = `ERROR: field expression %s contained unknown field %s
`
)

var objectCommand = &cobra.Command{
	Use:   "object",
	Short: "Manipulate object information",
}

func init() {
	objectCommand.AddCommand(objectListCommand)
	objectCommand.AddCommand(objectGetCommand)
	objectCommand.AddCommand(objectCreateCommand)
	objectCommand.AddCommand(objectDeleteCommand)
}

func buildObjectFilters() []*pb.ObjectFilter {
	filters := make([]*pb.ObjectFilter, 0)
	// Each --filter <field expression> supplied by the user will have one or
	// more $field=$value segments to it, separated by spaces. Split those
	// $field=$value pairs up and evaluate each $field and $value string for
	// fitness
	for _, f := range cliFilters {
		fieldExprs := strings.Fields(f)
		filter := &pb.ObjectFilter{}
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
				filter.Type = &pb.ObjectTypeFilter{
					Search:    value,
					UsePrefix: usePrefix,
				}
			case "project":
				filter.Project = value
			case "uuid":
				filter.Uuid = value
			case "name":
				filter.Name = value
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

func printObject(obj *pb.Object) {
	fmt.Printf("Partition:   %s\n", obj.Partition)
	fmt.Printf("Type:        %s\n", obj.Type)
	fmt.Printf("UUID:        %s\n", obj.Uuid)
	fmt.Printf("Name:        %s\n", obj.Name)
	if obj.Project != "" {
		fmt.Printf("Project:     %s\n", obj.Project)
	}
	if obj.Properties != nil {
		fmt.Printf("Properties:\n")
		for _, prop := range obj.Properties {
			fmt.Printf("   %s=%s\n", prop.Key, prop.Value)
		}
	}
	if obj.Tags != nil {
		tags := strings.Join(obj.Tags, ",")
		fmt.Printf("Tags:       %s\n", tags)
	}
}

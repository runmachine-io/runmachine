package commands

import (
	"fmt"
	"os"
	"strings"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/spf13/cobra"
)

const (
	usageProviderFilterOption = `optional filter to apply.

--filter <filter expression>

Multiple filters may be applied to the list operation. Each filter's field
expression is evaluated using an "AND" condition. Multiple filters are
evaluated using an "OR" condition.

The <filter expression> value is a whitespace-separated set of $field=$value
expressions to filter by. $field may be any of the following:

- partition: UUID or name of the partition the object belongs to
- type: code of the provider type (:see runm provider-type list)
- uuid: the UUID of the provider itself
- name: name of the provider

The $value should be an identifier or name for the $field. You can use an
asterisk (*) to indicate a prefix match. For example, to list all providers of
type "runm.compute" with names that start with the string "east", you would use
--filter "type=runm.compute name=east*"

Examples:

Find all runm.compute providers starting with "db":

--filter "type=runm.image name=db*"

Find all runm.compute providers OR runm.storage.block providers that are in a
partition called part0:

--filter "type=runm.compute partition=part0" \
--filter "type=runm.storage.block partition=part0"

Find a provider with a UUID of "f287341160ee4feba4012eb7f8125b82":

--filter "uuid=f287341160ee4feba4012eb7f8125b82"
`
)

var providerCommand = &cobra.Command{
	Use:   "provider",
	Short: "Manipulate provider information",
}

func init() {
	providerCommand.AddCommand(providerListCommand)
	providerCommand.AddCommand(providerGetCommand)
	providerCommand.AddCommand(providerCreateCommand)
}

func buildProviderFilters() []*pb.ProviderFilter {
	filters := make([]*pb.ProviderFilter, 0)
	// Each --filter <field expression> supplied by the user will have one or
	// more $field=$value segments to it, separated by spaces. Split those
	// $field=$value pairs up and evaluate each $field and $value string for
	// fitness
	for _, f := range cliFilters {
		fieldExprs := strings.Fields(f)
		filter := &pb.ProviderFilter{}
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
				filter.PartitionFilter = &pb.SearchFilter{
					Search:    value,
					UsePrefix: usePrefix,
				}
			case "type":
				filter.ProviderTypeFilter = &pb.SearchFilter{
					Search:    value,
					UsePrefix: usePrefix,
				}
			case "uuid":
			case "name":
				filter.PrimaryFilter = &pb.SearchFilter{
					Search:    value,
					UsePrefix: usePrefix,
				}
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

func printProvider(obj *pb.Provider) {
	fmt.Printf("Partition:     %s\n", obj.Partition)
	fmt.Printf("Provider Type: %s\n", obj.ProviderType)
	fmt.Printf("UUID:          %s\n", obj.Uuid)
	fmt.Printf("Name:          %s\n", obj.Name)
	fmt.Printf("Generation:    %d\n", obj.Generation)
	if obj.ParentUuid != "" {
		fmt.Printf("Parent:        %s\n", obj.ParentUuid)
	}
	if obj.Properties != nil {
		fmt.Printf("Properties:\n")
		for _, prop := range obj.Properties {
			fmt.Printf("   %s=%s\n", prop.Key, prop.Value)
		}
	}
	if obj.Tags != nil {
		tags := strings.Join(obj.Tags, ",")
		fmt.Printf("Tags:        %s\n", tags)
	}
}

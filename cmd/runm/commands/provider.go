package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	pb "github.com/runmachine-io/runmachine/proto"
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

If $field is not one of the above, the filter will be done on a property with
key $field, and the property's value should match $value. If the =$value part
of the filter expression is missing, the filter will return any providers
having a property with key $field regardless of the property's value.

The $value should be an identifier or name for the $field. You can use an
asterisk (*) to indicate a prefix match.

Examples:

Find all runm.compute providers starting with "db":

--filter "type=runm.compute name=db*"

Find all runm.compute providers OR runm.storage.block providers that are in a
partition called part0:

--filter "type=runm.compute partition=part0" \
--filter "type=runm.storage.block partition=part0"

Find a provider with a UUID of "f287341160ee4feba4012eb7f8125b82":

--filter "uuid=f287341160ee4feba4012eb7f8125b82"

Find all providers with the "location.site" property equal to "us-east":

--filter "location.site=us-east"

Find all providers having properties with both the "cpu.model" and "cpu.speed"
property keys associated with it:

--filter "cpu.model cpu.speed"

Find all providers having a "cpu.model" property with a value of "Intel Core
i7" OR providers having a "cpu.model" property with values beginning with the
string "AMD":

--filter 'cpu.model="Intel Core i7"' \
--filter "cpu.model=AMD*"
`
)

var providerCommand = &cobra.Command{
	Use:   "provider",
	Short: "Manipulate provider information",
}

func init() {
	providerCommand.AddCommand(providerDefinitionCommand)
	providerCommand.AddCommand(providerListCommand)
	providerCommand.AddCommand(providerGetCommand)
	providerCommand.AddCommand(providerCreateCommand)
	providerCommand.AddCommand(providerDeleteCommand)
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
		reqPropItems := make([]*pb.Property, 0)
		reqPropKeys := make([]string, 0)
		for _, fieldExpr := range fieldExprs {
			kvs := strings.SplitN(fieldExpr, "=", 2)
			field := kvs[0]
			if len(kvs) == 1 {
				// The user supplied something like --filter "cpu.model" which
				// indicates to filter for objects that have a property with
				// key "cpu.model" associated with it
				reqPropKeys = append(reqPropKeys, field)
				continue
			}
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
				// All other $fields are property key filters...
				reqPropItems = append(
					reqPropItems,
					&pb.Property{Key: field, Value: value},
				)
			}
		}
		if len(reqPropItems) > 0 || len(reqPropKeys) > 0 {
			filter.PropertyFilter = &pb.PropertyFilter{
				RequireItems: reqPropItems,
				RequireKeys:  reqPropKeys,
			}
		}
		filters = append(filters, filter)
	}
	return filters
}

func printProvider(obj *pb.Provider) {
	fmt.Printf("Partition:     %s\n", obj.Partition.Uuid)
	fmt.Printf("Provider Type: %s\n", obj.ProviderType.Code)
	fmt.Printf("UUID:          %s\n", obj.Uuid)
	fmt.Printf("Name:          %s\n", obj.Name)
	fmt.Printf("Generation:    %d\n", obj.Generation)
	if obj.Parent != nil {
		if obj.Parent.Uuid != "" {
			fmt.Printf("Parent:        %s\n", obj.Parent.Uuid)
		}
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

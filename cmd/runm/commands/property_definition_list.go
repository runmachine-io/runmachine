package commands

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/context"

	"github.com/olekukonko/tablewriter"
	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
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
`
)

var (
	// CLI-provided set of --filter options
	cliPropertyDefinitionFilters = []string{}
)

var propertyDefinitionListCommand = &cobra.Command{
	Use:   "list",
	Short: "List information about property definitions",
	Run:   propertyDefinitionList,
}

func setupPropertyDefinitionListFlags() {
	propertyDefinitionListCommand.Flags().StringArrayVarP(
		&cliPropertyDefinitionFilters,
		"filter", "f",
		nil,
		usagePropertyDefinitionFilterOption,
	)
}

func init() {
	setupPropertyDefinitionListFlags()
}

func buildPropertyDefinitionFilters() []*pb.PropertyDefinitionFilter {
	filters := make([]*pb.PropertyDefinitionFilter, 0)
	// Each --filter <field expression> supplied by the user will have one or
	// more $field=$value segments to it, separated by spaces. Split those
	// $field=$value pairs up and evaluate each $field and $value string for
	// fitness
	for _, f := range cliPropertyDefinitionFilters {
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
				filter.Type = &pb.ObjectTypeFilter{
					Search:    value,
					UsePrefix: usePrefix,
				}
			case "key":
				filter.Search = value
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

func propertyDefinitionList(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.PropertyDefinitionListRequest{
		Session: getSession(),
		Any:     buildPropertyDefinitionFilters(),
	}
	stream, err := client.PropertyDefinitionList(context.Background(), req)
	exitIfConnectErr(err)

	msgs := make([]*pb.PropertyDefinition, 0)
	for {
		role, err := stream.Recv()
		if err == io.EOF {
			break
		}
		exitIfError(err)
		msgs = append(msgs, role)
	}
	if len(msgs) == 0 {
		exitNoRecords()
	}
	headers := []string{
		"Partition",
		"Type",
		"Key",
		"Required?",
	}
	rows := make([][]string, len(msgs))
	for x, obj := range msgs {
		rows[x] = []string{
			obj.Partition,
			obj.Type,
			obj.Key,
			strconv.FormatBool(obj.IsRequired),
		}
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.Render()
}

package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/net/context"

	"github.com/olekukonko/tablewriter"
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
- object: UUID or name of the object

The $value should be an identifier or name for the $field. You can use an
asterisk (*) to indicate a prefix match. For example, to list all objects of
type "runm.machine" with names that start with the string "east", you would use
--filter "type=runm.machine object=east*"

Examples:

Find all runm.image objects starting with "db" in the "admin" project:

--filter "type=runm.image object=db* project=admin"

Find all runm.machine objects OR runm.image objects that are a partition called
part0:

--filter "type=runm.machine --partition=part0" \
--filter "type=runm.image partition=part0"

Find any object in the "admin" project that has a name starting with "db":

--filter "project=admin object=db*"
`
	errMsgFieldExprFormat = `ERROR: field expression %s expected to be in the form $field=$value
`
	errMsgUnknownFieldInFieldExpr = `ERROR: field expression %s contained unknown field %s
`
)

var (
	// CLI-provided set of --filter options
	cliObjectFilters = []string{}
)

var objectListCommand = &cobra.Command{
	Use:   "list",
	Short: "List information about objects",
	Run:   objectList,
}

func setupObjectListFlags() {
	objectListCommand.Flags().StringArrayVarP(
		&cliObjectFilters,
		"filter", "f",
		nil,
		usageObjectFilterOption,
	)
}

func init() {
	setupObjectListFlags()
}

func buildObjectFilters() []*pb.ObjectFilter {
	filters := make([]*pb.ObjectFilter, 0)
	// Each --filter <field expression> supplied by the user will have one or
	// more $field=$value segments to it, separated by spaces. Split those
	// $field=$value pairs up and evaluate each $field and $value string for
	// fitness
	for _, f := range cliObjectFilters {
		fieldExprs := strings.Fields(f)
		filter := &pb.ObjectFilter{}
		for _, fieldExpr := range fieldExprs {
			kvs := strings.SplitN(fieldExpr, "=", 1)
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
			case "project":
				filter.Project = value
			case "object":
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

func objectList(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.ObjectListRequest{
		Session: getSession(),
		Any:     buildObjectFilters(),
	}
	stream, err := client.ObjectList(context.Background(), req)
	exitIfConnectErr(err)

	msgs := make([]*pb.Object, 0)
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
		"UUID",
		"Name",
		"Project",
	}
	rows := make([][]string, len(msgs))
	for x, obj := range msgs {
		rows[x] = []string{
			obj.Partition,
			obj.ObjectType,
			obj.Uuid,
			obj.Name,
			obj.Project,
		}
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.Render()
}

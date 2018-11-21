package commands

import (
	"io"
	"os"
	"strings"

	"golang.org/x/net/context"

	"github.com/olekukonko/tablewriter"
	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

const (
	usageObjectTypeFilterOption = `optional filter to apply.

The filter value is the object type code to filter on. You can use an asterisk
(*) to indicate a prefix match. For example, to list all object types that
start with the string "runm", you would use --filter runm*
`
)

var (
	// CLI-provided set of --filter options
	cliObjectTypeFilters = []string{}
)

var objectTypeListCommand = &cobra.Command{
	Use:   "list",
	Short: "List information about object types",
	Run:   objectTypeList,
}

func setupObjectTypeListFlags() {
	objectTypeListCommand.Flags().StringArrayVarP(
		&cliObjectTypeFilters,
		"filter", "f",
		nil,
		usageObjectTypeFilterOption,
	)
}

func init() {
	setupObjectTypeListFlags()
}

func buildObjectTypeFilters() []*pb.ObjectTypeFilter {
	filters := make([]*pb.ObjectTypeFilter, 0)
	for _, f := range cliObjectTypeFilters {
		usePrefix := false
		if strings.HasSuffix(f, "*") {
			usePrefix = true
			f = strings.TrimRight(f, "*")
		}
		filters = append(
			filters,
			&pb.ObjectTypeFilter{
				Search:    f,
				UsePrefix: usePrefix,
			},
		)
	}
	return filters
}

func objectTypeList(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.ObjectTypeListRequest{
		Session: getSession(),
		Any:     buildObjectTypeFilters(),
	}
	stream, err := client.ObjectTypeList(context.Background(), req)
	exitIfConnectErr(err)

	msgs := make([]*pb.ObjectType, 0)
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
		"Code",
		"Scope",
		"Description",
	}
	rows := make([][]string, len(msgs))
	for x, obj := range msgs {
		rows[x] = []string{
			obj.Code,
			obj.Scope.String(),
			obj.Description,
		}
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.Render()
}

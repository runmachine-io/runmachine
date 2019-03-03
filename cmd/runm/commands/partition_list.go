package commands

import (
	"io"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	usagePartitionFilterOption = `optional filter to apply.

The filter value is the partition UUID or name to filter on. You can use an
asterisk (*) to indicate a prefix match. For example, to list all partitions
with names that start with the string "east", you would use --filter east*
`
)

var (
	// CLI-provided set of --filter options
	cliPartitionFilters = []string{}
)

var partitionListCommand = &cobra.Command{
	Use:   "list",
	Short: "List information about partitions",
	Run:   partitionList,
}

func setupPartitionListFlags() {
	partitionListCommand.Flags().StringArrayVarP(
		&cliPartitionFilters,
		"filter", "f",
		nil,
		usagePartitionFilterOption,
	)
}

func init() {
	setupPartitionListFlags()
}

func buildPartitionFilters() []*pb.PartitionFilter {
	filters := make([]*pb.PartitionFilter, 0)
	for _, f := range cliPartitionFilters {
		usePrefix := false
		if strings.HasSuffix(f, "*") {
			usePrefix = true
			f = strings.TrimRight(f, "*")
		}
		filters = append(
			filters,
			&pb.PartitionFilter{
				PrimaryFilter: &pb.SearchFilter{
					Search:    f,
					UsePrefix: usePrefix,
				},
			},
		)
	}
	return filters
}

func partitionList(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)
	req := &pb.PartitionListRequest{
		Session: getSession(),
		Any:     buildPartitionFilters(),
	}
	stream, err := client.PartitionList(context.Background(), req)
	exitIfConnectErr(err)

	msgs := make([]*pb.Partition, 0)
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
		"UUID",
		"Name",
	}
	rows := make([][]string, len(msgs))
	for x, obj := range msgs {
		rows[x] = []string{
			obj.Uuid,
			obj.Name,
		}
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.Render()
}

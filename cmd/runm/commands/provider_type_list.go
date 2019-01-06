package commands

import (
	"io"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
)

const (
	usageProviderTypeFilterOption = `optional filter to apply.

The filter value is the provider type code to filter on. You can use an asterisk
(*) to indicate a prefix match. For example, to list all provider types that
start with the string "runm.storage", you would use --filter runm.storage*
`
)

var providerTypeListCommand = &cobra.Command{
	Use:   "list",
	Short: "List information about provider types",
	Run:   providerTypeList,
}

func setupProviderTypeListFlags() {
	providerTypeListCommand.Flags().StringArrayVarP(
		&cliFilters,
		"filter", "f",
		nil,
		usageProviderTypeFilterOption,
	)
}

func init() {
	setupProviderTypeListFlags()
}

func buildProviderTypeFilters() []*pb.ProviderTypeFilter {
	filters := make([]*pb.ProviderTypeFilter, 0)
	for _, f := range cliFilters {
		usePrefix := false
		if strings.HasSuffix(f, "*") {
			usePrefix = true
			f = strings.TrimRight(f, "*")
		}
		filters = append(
			filters,
			&pb.ProviderTypeFilter{
				Search:    f,
				UsePrefix: usePrefix,
			},
		)
	}
	return filters
}

func providerTypeList(cmd *cobra.Command, args []string) {
	conn := apiConnect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)
	req := &pb.ProviderTypeListRequest{
		Session: apiGetSession(),
		Any:     buildProviderTypeFilters(),
	}
	stream, err := client.ProviderTypeList(context.Background(), req)
	exitIfConnectErr(err)

	msgs := make([]*pb.ProviderType, 0)
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
		"Description",
	}
	rows := make([][]string, len(msgs))
	for x, obj := range msgs {
		rows[x] = []string{
			obj.Code,
			obj.Description,
		}
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.Render()
}

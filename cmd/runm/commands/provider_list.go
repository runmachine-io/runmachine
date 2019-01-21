package commands

import (
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	apipb "github.com/runmachine-io/runmachine/pkg/api/proto"
)

var providerListCommand = &cobra.Command{
	Use:   "list",
	Short: "List information about providers",
	Run:   providerList,
}

func setupProviderListFlags() {
	providerListCommand.Flags().StringArrayVarP(
		&cliFilters,
		"filter", "f",
		nil,
		usageProviderFilterOption,
	)
}

func init() {
	setupProviderListFlags()
}

func providerList(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := apipb.NewRunmAPIClient(conn)
	req := &apipb.ProviderListRequest{
		Session: getSession(),
		Any:     buildProviderFilters(),
	}
	stream, err := client.ProviderList(context.Background(), req)
	exitIfConnectErr(err)

	msgs := make([]*apipb.Provider, 0)
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
		"Provider Type",
		"UUID",
		"Name",
	}
	rows := make([][]string, len(msgs))
	for x, obj := range msgs {
		rows[x] = []string{
			obj.Partition,
			obj.ProviderType,
			obj.Uuid,
			obj.Name,
		}
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.Render()
}

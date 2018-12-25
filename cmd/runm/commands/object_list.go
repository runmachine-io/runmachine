package commands

import (
	"io"
	"os"

	"golang.org/x/net/context"

	"github.com/olekukonko/tablewriter"
	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var objectListCommand = &cobra.Command{
	Use:   "list",
	Short: "List information about objects",
	Run:   objectList,
}

func setupObjectListFlags() {
	objectListCommand.Flags().StringArrayVarP(
		&cliFilters,
		"filter", "f",
		nil,
		usageObjectFilterOption,
	)
}

func init() {
	setupObjectListFlags()
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
			obj.Type,
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

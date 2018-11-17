package commands

import (
	"io"
	"os"

	"golang.org/x/net/context"

	"github.com/olekukonko/tablewriter"
	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var propertySchemaListCommand = &cobra.Command{
	Use:   "list",
	Short: "List information about property schemas",
	Run:   propertySchemaList,
}

func propertySchemaList(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.PropertySchemaListRequest{
		Session: getSession(),
	}
	stream, err := client.PropertySchemaList(context.Background(), req)
	exitIfConnectErr(err)

	msgs := make([]*pb.PropertySchema, 0)
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
		"Object Type",
		"Key",
		"Schema",
	}
	rows := make([][]string, len(msgs))
	for x, obj := range msgs {
		rows[x] = []string{
			obj.Partition,
			obj.ObjectType,
			obj.Key,
			obj.Schema,
		}
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.Render()
}

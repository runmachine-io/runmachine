package commands

import (
	"fmt"
	"io"
	"os"
	"strconv"

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
	filters := &pb.PropertySchemaListFilters{}
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.PropertySchemaListRequest{
		Session: &pb.Session{},
		Filters: filters,
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
		"Version",
		"Schema",
	}
	rows := make([][]string, len(msgs))
	for x, msg := range msgs {
		partition := ""
		if msg.Partition != nil {
			partition = fmt.Sprintf(
				"%s",
				msg.Partition.DisplayName,
			)
		}
		objType := ""
		if msg.ObjectType != nil {
			objType = fmt.Sprintf(
				"%s",
				msg.ObjectType.Code,
			)
		}
		rows[x] = []string{
			partition,
			objType,
			msg.Key,
			strconv.Itoa(int(msg.Version)),
			msg.Schema,
		}
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.Render()
}

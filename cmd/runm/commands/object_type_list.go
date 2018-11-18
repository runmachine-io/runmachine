package commands

import (
	"io"
	"os"

	"golang.org/x/net/context"

	"github.com/olekukonko/tablewriter"
	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var objectTypeListCommand = &cobra.Command{
	Use:   "list",
	Short: "List information about object types",
	Run:   objectTypeList,
}

func objectTypeList(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.ObjectTypeListRequest{
		Session: getSession(),
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

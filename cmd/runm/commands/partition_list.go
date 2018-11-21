package commands

import (
	"io"
	"os"

	"golang.org/x/net/context"

	"github.com/olekukonko/tablewriter"
	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var partitionListCommand = &cobra.Command{
	Use:   "list",
	Short: "List information about partitions",
	Run:   partitionList,
}

func partitionList(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.PartitionListRequest{
		Session: getSession(),
		// TODO(jaypipes): Allow filtering on name/UUID of partition (with name
		// prefix?)
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

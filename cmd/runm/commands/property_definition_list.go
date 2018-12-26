package commands

import (
	"io"
	"os"
	"strconv"

	"golang.org/x/net/context"

	"github.com/olekukonko/tablewriter"
	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var propertyDefinitionListCommand = &cobra.Command{
	Use:   "list",
	Short: "List information about property definitions",
	Run:   propertyDefinitionList,
}

func setupPropertyDefinitionListFlags() {
	propertyDefinitionListCommand.Flags().StringArrayVarP(
		&cliFilters,
		"filter", "f",
		nil,
		usagePropertyDefinitionFilterOption,
	)
}

func init() {
	setupPropertyDefinitionListFlags()
}

func propertyDefinitionList(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.PropertyDefinitionListRequest{
		Session: getSession(),
		Any:     buildPropertyDefinitionFilters(),
	}
	stream, err := client.PropertyDefinitionList(context.Background(), req)
	exitIfConnectErr(err)

	msgs := make([]*pb.PropertyDefinition, 0)
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
		"UUID",
		"Required?",
	}
	rows := make([][]string, len(msgs))
	for x, obj := range msgs {
		rows[x] = []string{
			obj.Partition,
			obj.ObjectType,
			obj.Key,
			obj.Uuid,
			strconv.FormatBool(obj.IsRequired),
		}
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.Render()
}

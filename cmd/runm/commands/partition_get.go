package commands

import (
	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
)

var partitionGetCommand = &cobra.Command{
	Use:   "get <search>",
	Short: "Show information for a single partition",
	Args:  cobra.ExactArgs(1),
	Run:   partitionGet,
}

func partitionGet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)

	session := getSession()

	req := &pb.PartitionGetRequest{
		Session: session,
		Filter: &pb.PartitionFilter{
			PrimaryFilter: &pb.SearchFilter{
				Search: args[0],
			},
		},
	}
	obj, err := client.PartitionGet(context.Background(), req)
	exitIfError(err)
	printPartition(obj)
}

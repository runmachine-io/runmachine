package commands

import (
	"fmt"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
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

	client := pb.NewRunmMetadataClient(conn)

	session := getSession()

	req := &pb.PartitionGetRequest{
		Session: session,
		Filter: &pb.PartitionFilter{
			Search: args[0],
		},
	}
	obj, err := client.PartitionGet(context.Background(), req)
	exitIfError(err)
	fmt.Printf("UUID: %s\n", obj.Uuid)
	fmt.Printf("Name: %s\n", obj.Name)
}

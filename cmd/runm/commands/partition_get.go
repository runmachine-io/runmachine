package commands

import (
	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	apipb "github.com/runmachine-io/runmachine/pkg/api/proto"
)

var partitionGetCommand = &cobra.Command{
	Use:   "get <search>",
	Short: "Show information for a single partition",
	Args:  cobra.ExactArgs(1),
	Run:   partitionGet,
}

func partitionGet(cmd *cobra.Command, args []string) {
	conn := apiConnect()
	defer conn.Close()

	client := apipb.NewRunmAPIClient(conn)

	session := apiGetSession()

	req := &apipb.PartitionGetRequest{
		Session: session,
		Filter: &apipb.PartitionFilter{
			Search: args[0],
		},
	}
	obj, err := client.PartitionGet(context.Background(), req)
	exitIfError(err)
	printPartition(obj)
}

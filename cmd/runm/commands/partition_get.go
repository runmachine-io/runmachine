package commands

import (
	"fmt"

	"golang.org/x/net/context"

	apipb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/spf13/cobra"
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
	fmt.Printf("UUID: %s\n", obj.Uuid)
	fmt.Printf("Name: %s\n", obj.Name)
}

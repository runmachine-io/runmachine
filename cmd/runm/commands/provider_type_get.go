package commands

import (
	"golang.org/x/net/context"

	"github.com/spf13/cobra"

	pb "github.com/runmachine-io/runmachine/proto"
)

var providerTypeGetCommand = &cobra.Command{
	Use:   "get <code>",
	Short: "Show information for a single provider type",
	Args:  cobra.ExactArgs(1),
	Run:   providerTypeGet,
}

func providerTypeGet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)

	session := getSession()

	req := &pb.ProviderTypeGetRequest{
		Session: session,
		Filter: &pb.ProviderTypeFilter{
			Search:    args[0],
			UsePrefix: false,
		},
	}
	obj, err := client.ProviderTypeGet(context.Background(), req)
	exitIfError(err)
	printProviderType(obj)
}

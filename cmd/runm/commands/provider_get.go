package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
)

const (
	usageProviderGet = `Show information for a single provider

Specify a single CLI argument with the UUID or name of the provider you wish to
show:

  runm provider get 4f4f54c9bfb44cce9a02d4daf6f79ea3

or

  runm provider get us-east1.row1.rack23.node14

NOTE: when using the second form, with the provider's name instead of UUID, the
user's session partition is used to find the provider.
`
)

var providerGetCommand = &cobra.Command{
	Use:   "get <search>",
	Short: "Show information for a single provider",
	Run:   providerGet,
	Long:  usageProviderGet,
}

func providerGet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)

	session := apiGetSession()

	var filter *pb.ProviderFilter

	if len(args) != 1 {
		fmt.Fprintf(
			os.Stderr,
			"Error: please provide a single argument: either specify a UUID "+
				"or a name for the provider to show\n",
		)
		cmd.Help()
		os.Exit(1)
	}

	filter = &pb.ProviderFilter{
		Search:    args[0],
		UsePrefix: false,
	}
	obj, err := client.ProviderGet(
		context.Background(),
		&pb.ProviderGetRequest{
			Session: session,
			Filter:  filter,
		},
	)
	exitIfError(err)
	printProvider(obj)
}

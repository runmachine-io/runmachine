package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
)

const (
	usageProviderDefinitionGet = `Show information for a provider definition

Specify a single CLI argument with the UUID or name of the partition you wish
to show the provider definition for:

  runm provider definition get 4f4f54c9bfb44cce9a02d4daf6f79ea3

or

  runm provider definition get part0
`
)

var providerDefinitionGetCommand = &cobra.Command{
	Use:   "get <search>",
	Short: "Show information for a provider definition",
	Run:   providerDefinitionGet,
	Long:  usageProviderDefinitionGet,
}

func providerDefinitionGet(cmd *cobra.Command, args []string) {
	conn := apiConnect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)

	session := apiGetSession()

	var filter *pb.ProviderDefinitionFilter

	if len(args) != 1 {
		fmt.Fprintf(
			os.Stderr,
			"Error: please provide a single argument: either specify a UUID "+
				"or a name for the partition whose provider definition you "+
				"wish to show\n",
		)
		cmd.Help()
		os.Exit(1)
	}

	filter = &pb.ProviderDefinitionFilter{
		PartitionFilter: &pb.SearchFilter{
			Search:    args[0],
			UsePrefix: false,
		},
	}
	obj, err := client.ProviderDefinitionGet(
		context.Background(),
		&pb.ProviderDefinitionGetRequest{
			Session: session,
			Filter:  filter,
		},
	)
	exitIfError(err)
	printProviderDefinition(obj)
}

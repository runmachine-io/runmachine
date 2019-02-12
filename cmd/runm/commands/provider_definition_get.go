package commands

import (
	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
)

const (
	usageProviderDefinitionGet = `Show the definition for providers

Specifying the --global CLI option will show the global default provider
definition:

  runm provider definition get --global

To show the definition for providers in a specific partition, if an admin has
overridden the definition for providers in that partition, use the --partition
CLI option:

  runm provider definition get --partition part0

The --type CLI option can be used to return a provider definition that has been
set for a specific provider type:

  runm provider definition get --type "runm.compute"

HINT: To see the list of valid provider types:

  runm provider-type list

NOTE: Specifying neither --global nor --partition CLI options will return the
definition for providers in the user's session partition *or* the global
default if no override definition for providers in that partition has been set.

In other words, running this command with no --global or --partition CLI option
will show the exact definition that will be used to validate provider input
data if the user calls the runm provider create command and the user's session
partition is used for the supplied provider input data.

The --type CLI option may be used together with either the --global or
--partition CLI option. If neither the --global nor --partition CLI options are
supplied and the --type CLI option is used, then the provider definition set
for the user's session partition and the supplied provider type is returned.
`
)

var providerDefinitionGetCommand = &cobra.Command{
	Use:   "get",
	Short: "Show information for a provider definition",
	Run:   providerDefinitionGet,
	Long:  usageProviderDefinitionGet,
}

func setupProviderDefinitionGetFlags() {
	providerDefinitionGetCommand.Flags().BoolVarP(
		&cliProviderDefinitionGlobal,
		"global", "g",
		false,
		"Show the global default definition for providers.",
	)
	providerDefinitionGetCommand.Flags().StringVarP(
		&cliProviderDefinitionPartition,
		"partition", "",
		"",
		"Optional partition identifier.",
	)
	providerDefinitionGetCommand.Flags().StringVarP(
		&cliProviderDefinitionType,
		"type", "t",
		"",
		"Optional provider type.",
	)
}

func init() {
	setupProviderDefinitionGetFlags()
}

func providerDefinitionGet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)

	session := getSession()

	var tryGlobalFallback bool = false
	var argPartition string
	if cliProviderDefinitionGlobal {
		argPartition = ""
	} else {
		if cliProviderDefinitionPartition == "" {
			argPartition = session.Partition
			tryGlobalFallback = true
		} else {
			argPartition = cliProviderDefinitionPartition
		}
	}

	obj, err := client.ProviderDefinitionGet(
		context.Background(),
		&pb.ProviderDefinitionGetRequest{
			Session:      session,
			Partition:    argPartition,
			ProviderType: cliProviderDefinitionType,
		},
	)
	if errIsNotFound(err) {
		if !tryGlobalFallback {
			exitIfError(err)
		}
		obj, err = client.ProviderDefinitionGet(
			context.Background(),
			&pb.ProviderDefinitionGetRequest{
				Session:      session,
				Partition:    "",
				ProviderType: cliProviderDefinitionType,
			},
		)
		exitIfError(err)
	}
	printObjectDefinition(obj)
}

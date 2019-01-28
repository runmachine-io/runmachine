package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
)

const (
	usageProviderDefinitionSet = `Set the definition for providers

Specifying the --global CLI option will set the global default provider
definition:

  runm provider definition set --global

To override the definition for providers in a specific partition, use the
--partition CLI option:

  runm provider definition set --partition part0

The --type CLI option can be used to override a provider definition for a
specific provider type:

  runm provider definition set --type "runm.compute"

HINT: To see the list of valid provider types:

  runm provider-type list

NOTE: Specifying neither --global or --partition CLI options will set the
definition for providers in the user's session partition.

In other words, running this command with no --global or --partition CLI option
will override the definition that will be used to validate provider input data
if the user calls the runm provider create command and the user's session
partition is used for the supplied provider input data.

The --type CLI option may be used together with either the --global or
--partition CLI option. If neither the --global nor --partition CLI options are
supplied and the --type CLI option is used, then the provider definition for
the user's session partition and the supplied provider type is set.
`
)

var providerDefinitionSetCommand = &cobra.Command{
	Use:   "set",
	Short: "Define the schema for providers",
	Run:   providerDefinitionSet,
	Long:  usageProviderDefinitionSet,
}

func setupProviderDefinitionSetFlags() {
	providerDefinitionSetCommand.Flags().BoolVarP(
		&cliProviderDefinitionGlobal,
		"global", "g",
		false,
		"Set the global default definition for providers.",
	)
	providerDefinitionSetCommand.Flags().StringVarP(
		&cliProviderDefinitionPartition,
		"partition", "",
		"",
		"Identifier of partition to set an override provider definition for.",
	)
	providerDefinitionSetCommand.Flags().StringVarP(
		&cliProviderDefinitionType,
		"type", "t",
		"",
		"Optional provider type.",
	)
	providerDefinitionSetCommand.Flags().StringVarP(
		&cliObjectDocPath,
		"file", "f",
		"",
		"optional filepath to YAML document to send.",
	)
}

func init() {
	setupProviderDefinitionSetFlags()
}

func providerDefinitionSet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)
	session := getSession()

	var argPartition string
	if cliProviderDefinitionGlobal {
		argPartition = ""
	} else {
		if cliProviderDefinitionPartition == "" {
			argPartition = session.Partition
		} else {
			argPartition = cliProviderDefinitionPartition
		}
	}

	req := &pb.ProviderDefinitionSetRequest{
		Session:      session,
		Format:       pb.PayloadFormat_YAML,
		Payload:      readInputDocumentOrExit(),
		Partition:    argPartition,
		ProviderType: cliProviderDefinitionType,
	}

	resp, err := client.ProviderDefinitionSet(context.Background(), req)
	exitIfError(err)
	obj := resp.ObjectDefinition
	if !quiet {
		fmt.Printf("ok\n")
		if verbose {
			printObjectDefinition(obj)
		}
	}
}

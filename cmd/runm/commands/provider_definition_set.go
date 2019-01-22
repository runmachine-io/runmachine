package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
)

var providerDefinitionSetCommand = &cobra.Command{
	Use:   "set",
	Short: "Define the schema for providers",
	Run:   providerDefinitionSet,
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
		Session:   session,
		Format:    pb.PayloadFormat_YAML,
		Payload:   readInputDocumentOrExit(),
		Partition: argPartition,
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

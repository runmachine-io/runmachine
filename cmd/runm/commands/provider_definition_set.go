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
	conn := apiConnect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)
	req := &pb.CreateRequest{
		Session: apiGetSession(),
		Format:  pb.PayloadFormat_YAML,
		Payload: readInputDocumentOrExit(),
	}

	resp, err := client.ProviderDefinitionSet(context.Background(), req)
	exitIfError(err)
	obj := resp.ProviderDefinition
	if !quiet {
		fmt.Printf("ok\n")
		if verbose {
			printProviderDefinition(obj)
		}
	}
}

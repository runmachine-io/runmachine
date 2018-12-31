package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

var propertyDefinitionSetCommand = &cobra.Command{
	Use:   "set",
	Short: "Create or update a property definition",
	Run:   propertyDefinitionSet,
}

func setupPropertyDefinitionSetFlags() {
	propertyDefinitionSetCommand.Flags().StringVarP(
		&cliObjectDocPath,
		"file", "f",
		"",
		"optional filepath to YAML document to send.",
	)
}

func init() {
	setupPropertyDefinitionSetFlags()
}

func propertyDefinitionSet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.PropertyDefinitionSetRequest{
		Session: getSession(),
		Format:  pb.PayloadFormat_YAML,
		Payload: readInputDocumentOrExit(),
	}

	resp, err := client.PropertyDefinitionSet(context.Background(), req)
	exitIfError(err)
	obj := resp.PropertyDefinition
	if !quiet {
		fmt.Printf("ok\n")
		if verbose {
			printPropertyDefinition(obj)
		}
	}
}

package commands

import (
	"fmt"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
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
			fmt.Printf("Partition:    %s\n", obj.Partition)
			fmt.Printf("Type:         %s\n", obj.Type)
			fmt.Printf("Key:          %s\n", obj.Key)
			fmt.Printf("Schema:\n%s\n", obj.Schema)
		}
	}
}

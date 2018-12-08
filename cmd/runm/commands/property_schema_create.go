package commands

import (
	"fmt"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var propertySchemaCreateCommand = &cobra.Command{
	Use:   "create",
	Short: "Create a new property schema",
	Run:   propertySchemaCreate,
}

func setupPropertySchemaCreateFlags() {
	propertySchemaCreateCommand.Flags().StringVarP(
		&cliObjectDocPath,
		"file", "f",
		"",
		"optional filepath to YAML document to send.",
	)
}

func init() {
	setupPropertySchemaCreateFlags()
}

func propertySchemaCreate(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.PropertySchemaSetRequest{
		Session: getSession(),
		Format:  pb.PayloadFormat_YAML,
		Payload: readInputDocumentOrExit(),
	}

	resp, err := client.PropertySchemaSet(context.Background(), req)
	exitIfError(err)
	obj := resp.PropertySchema
	if !quiet {
		fmt.Printf("Successfully created property schema\n")
		if verbose {
			fmt.Printf("Partition:    %s\n", obj.Partition)
			fmt.Printf("Type:         %s\n", obj.Type)
			fmt.Printf("Key:          %s\n", obj.Key)
			fmt.Printf("Schema:\n%s\n", obj.Schema)
		}
	}
}

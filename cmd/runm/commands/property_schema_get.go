package commands

import (
	"fmt"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var (
	// partition override. if empty, we use the session's partition
	propSchemaGetPartition string
)

var propertySchemaGetCommand = &cobra.Command{
	Use:   "get <object_type> <key>",
	Short: "Show information for a single property schema",
	Args:  cobra.ExactArgs(2),
	Run:   propertySchemaGet,
}

func setupPropertySchemaGetFlags() {
	propertySchemaGetCommand.Flags().StringVarP(
		&propSchemaGetPartition,
		"partition", "p",
		"",
		"optional partition in which to look for the property schema.",
	)
}

func init() {
	setupPropertySchemaGetFlags()
}

func propertySchemaGet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)

	session := getSession()

	req := &pb.PropertySchemaGetRequest{
		Session:   session,
		Partition: propSchemaGetPartition,
		Type:      args[0],
		Key:       args[1],
	}
	obj, err := client.PropertySchemaGet(context.Background(), req)
	exitIfError(err)
	fmt.Printf("Partition:    %s\n", obj.Partition)
	fmt.Printf("Type:         %s\n", obj.Type)
	fmt.Printf("Key:          %s\n", obj.Key)
	fmt.Printf("Schema:\n%s\n", obj.Schema)
}

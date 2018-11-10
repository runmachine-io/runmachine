package commands

import (
	"fmt"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var propertySchemaGetCommand = &cobra.Command{
	Use:   "get <object_type> <key>",
	Short: "Show information for a single property schema",
	Args:  cobra.ExactArgs(2),
	Run:   propertySchemaGet,
}

func propertySchemaGet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.PropertySchemaGetRequest{
		Session: getSession(),
		ObjectType: &pb.ObjectType{
			Code: args[0],
		},
		Key: args[1],
	}
	obj, err := client.PropertySchemaGet(context.Background(), req)
	exitIfError(err)
	fmt.Printf("Partition:    %s\n", obj.Partition.Uuid)
	fmt.Printf("Object type:  %s\n", obj.ObjectType.Code)
	fmt.Printf("Key:          %s\n", obj.Key)
	fmt.Printf("Version:      %d\n", obj.Version)
	fmt.Printf("Schema:\n%s\n", obj.Schema)
}

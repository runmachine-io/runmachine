package commands

import (
	"fmt"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var propertySchemaCommand = &cobra.Command{
	Use:   "property-schema",
	Short: "Manipulate property schema information",
}

func init() {
	propertySchemaCommand.AddCommand(propertySchemaListCommand)
	propertySchemaCommand.AddCommand(propertySchemaGetCommand)
	propertySchemaCommand.AddCommand(propertySchemaCreateCommand)
}

func printPropertySchema(obj *pb.PropertySchema) {
	fmt.Printf("Partition:    %s\n", obj.Partition)
	fmt.Printf("Type:         %s\n", obj.Type)
	fmt.Printf("Key:          %s\n", obj.Key)
	fmt.Printf("Schema:\n%s\n", obj.Schema)
}

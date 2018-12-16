package commands

import (
	"fmt"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var propertyDefinitionCommand = &cobra.Command{
	Use:   "property-definition",
	Short: "Manipulate property definition information",
}

func init() {
	propertyDefinitionCommand.AddCommand(propertyDefinitionListCommand)
	propertyDefinitionCommand.AddCommand(propertyDefinitionGetCommand)
	propertyDefinitionCommand.AddCommand(propertyDefinitionCreateCommand)
}

func printPropertyDefinition(obj *pb.PropertyDefinition) {
	fmt.Printf("Partition:    %s\n", obj.Partition)
	fmt.Printf("Type:         %s\n", obj.Type)
	fmt.Printf("Key:          %s\n", obj.Key)
	fmt.Printf("Schema:\n%s\n", obj.Schema)
}

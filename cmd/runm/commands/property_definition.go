package commands

import (
	"fmt"
	"strconv"

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
	propertyDefinitionCommand.AddCommand(propertyDefinitionSetCommand)
	propertyDefinitionCommand.AddCommand(propertyDefinitionDeleteCommand)
}

func printPropertyDefinition(obj *pb.PropertyDefinition) {
	fmt.Printf("Partition:    %s\n", obj.Partition)
	fmt.Printf("Type:         %s\n", obj.Type)
	fmt.Printf("Key:          %s\n", obj.Key)
	fmt.Printf("Required:     %s\n", strconv.FormatBool(obj.IsRequired))
	fmt.Printf("Schema:\n%s", obj.Schema)
}

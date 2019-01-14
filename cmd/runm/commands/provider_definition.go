package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
)

var providerDefinitionCommand = &cobra.Command{
	Use:   "definition",
	Short: "Manipulate provider definitions",
}

func init() {
	providerDefinitionCommand.AddCommand(providerDefinitionSetCommand)
}

func printProviderDefinition(obj *pb.ProviderDefinition) {
	fmt.Printf("Partition:    %s\n", obj.Partition)
	fmt.Printf("Schema:\n%s", obj.Schema)
}

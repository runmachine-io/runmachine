package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
)

var providerTypeCommand = &cobra.Command{
	Use:   "provider-type",
	Short: "Fetch provider type information",
}

func init() {
	providerTypeCommand.AddCommand(providerTypeGetCommand)
	providerTypeCommand.AddCommand(providerTypeListCommand)
}

func printProviderType(obj *pb.ProviderType) {
	fmt.Printf("Code:        %s\n", obj.Code)
	fmt.Printf("Description: %s\n", obj.Description)
}

package commands

import (
	"fmt"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/spf13/cobra"
)

var providerCommand = &cobra.Command{
	Use:   "provider",
	Short: "Manipulate provider information",
}

func init() {
	providerCommand.AddCommand(providerGetCommand)
}

func printProvider(obj *pb.Provider) {
	fmt.Printf("Partition:   %s\n", obj.Partition)
	fmt.Printf("Provider Type: %s\n", obj.ProviderType)
	fmt.Printf("UUID:        %s\n", obj.Uuid)
	fmt.Printf("Name:        %s\n", obj.Name)
	if obj.ParentUuid != "" {
		fmt.Printf("Parent:     %s\n", obj.ParentUuid)
	}
	// TODO(jaypipes): Add support for properties and tags
	//if obj.Properties != nil {
	//	fmt.Printf("Properties:\n")
	//	for _, prop := range obj.Properties {
	//		fmt.Printf("   %s=%s\n", prop.Key, prop.Value)
	//	}
	//}
	//if obj.Tags != nil {
	//	tags := strings.Join(obj.Tags, ",")
	//	fmt.Printf("Tags:        %s\n", tags)
	//}
}

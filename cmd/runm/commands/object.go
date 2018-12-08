package commands

import (
	"fmt"
	"strings"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var objectCommand = &cobra.Command{
	Use:   "object",
	Short: "Manipulate object information",
}

func init() {
	objectCommand.AddCommand(objectListCommand)
	objectCommand.AddCommand(objectGetCommand)
	objectCommand.AddCommand(objectCreateCommand)
	objectCommand.AddCommand(objectDeleteCommand)
}

func printObject(obj *pb.Object) {
	fmt.Printf("Partition:   %s\n", obj.Partition)
	fmt.Printf("Type:        %s\n", obj.Type)
	fmt.Printf("UUID:        %s\n", obj.Uuid)
	fmt.Printf("Name:        %s\n", obj.Name)
	if obj.Project != "" {
		fmt.Printf("Project:     %s\n", obj.Project)
	}
	if obj.Properties != nil {
		fmt.Printf("Properties:\n")
		for _, prop := range obj.Properties {
			fmt.Printf("   %s=%s\n", prop.Key, prop.Value)
		}
	}
	if obj.Tags != nil {
		tags := strings.Join(obj.Tags, ",")
		fmt.Printf("Tags:       %s\n", tags)
	}
}

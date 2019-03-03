package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	pb "github.com/runmachine-io/runmachine/proto"
)

var partitionCommand = &cobra.Command{
	Use:   "partition",
	Short: "Fetch partition information",
}

func init() {
	partitionCommand.AddCommand(partitionListCommand)
	partitionCommand.AddCommand(partitionGetCommand)
	partitionCommand.AddCommand(partitionCreateCommand)
}

func printPartition(obj *pb.Partition) {
	fmt.Printf("UUID: %s\n", obj.Uuid)
	fmt.Printf("Name: %s\n", obj.Name)
}

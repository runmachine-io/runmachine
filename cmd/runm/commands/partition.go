package commands

import (
	"fmt"

	apipb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/spf13/cobra"
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

func printPartition(obj *apipb.Partition) {
	fmt.Printf("UUID: %s\n", obj.Uuid)
	fmt.Printf("Name: %s\n", obj.Name)
}

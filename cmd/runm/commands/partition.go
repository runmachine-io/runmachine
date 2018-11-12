package commands

import (
	"github.com/spf13/cobra"
)

var partitionCommand = &cobra.Command{
	Use:   "partition",
	Short: "Fetch partition information",
}

func init() {
	partitionCommand.AddCommand(partitionListCommand)
	partitionCommand.AddCommand(partitionGetCommand)
}

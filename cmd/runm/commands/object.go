package commands

import (
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
}

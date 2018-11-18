package commands

import (
	"github.com/spf13/cobra"
)

var objectTypeCommand = &cobra.Command{
	Use:   "object-type",
	Short: "Fetch object type information",
}

func init() {
	objectTypeCommand.AddCommand(objectTypeListCommand)
	objectTypeCommand.AddCommand(objectTypeGetCommand)
}

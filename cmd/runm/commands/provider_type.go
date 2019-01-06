package commands

import (
	"github.com/spf13/cobra"
)

var providerTypeCommand = &cobra.Command{
	Use:   "provider-type",
	Short: "Fetch provider type information",
}

func init() {
	providerTypeCommand.AddCommand(providerTypeListCommand)
}

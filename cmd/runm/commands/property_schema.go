package commands

import (
	"github.com/spf13/cobra"
)

var propertySchemaCommand = &cobra.Command{
	Use:   "property-schema",
	Short: "Manipulate property schema information",
}

func init() {
	propertySchemaCommand.AddCommand(propertySchemaListCommand)
}

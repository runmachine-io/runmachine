package commands

import (
	"github.com/spf13/cobra"
)

var propertySchemaCommand = &cobra.Command{
	Use:   "property-schema",
	Short: "Manipulate property schema information",
}

func init() {
	propertySchemaCommand.AddCommand(propertySchemaSetCommand)
	propertySchemaCommand.AddCommand(propertySchemaListCommand)
	propertySchemaCommand.AddCommand(propertySchemaGetCommand)
}

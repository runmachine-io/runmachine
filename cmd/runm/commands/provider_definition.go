package commands

import (
	"github.com/spf13/cobra"
)

var (
	cliProviderDefinitionGlobal    bool
	cliProviderDefinitionPartition string
	cliProviderDefinitionType      string
)

var providerDefinitionCommand = &cobra.Command{
	Use:   "definition",
	Short: "Manipulate provider definitions",
}

func init() {
	providerDefinitionCommand.AddCommand(providerDefinitionGetCommand)
	providerDefinitionCommand.AddCommand(providerDefinitionSetCommand)
}

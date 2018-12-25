package commands

import (
	"fmt"
	"os"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

const (
	usagePropertyDefinitionGet = `Show information for a single property definition

There are two call signatures to this command.

The first is to specify a single CLI argument that should be the UUID of the
property definition you wish to show:

  runm property-definition get 4f4f54c9bfb44cce9a02d4daf6f79ea3

The second is to specify a --filter option string that returns a single
property definition.

  runm property-definition get --filter "type=runm.image key=architecture"

Specifying a --filter option string that returns more than one property
definition will result in a MultipleRecordsFound error.

The --filter option string will ignored when also supplying a <UUID> argument.
`
)

var propertyDefinitionGetCommand = &cobra.Command{
	Use:   "get [<UUID>]",
	Short: "Show information for a single property definition",
	Run:   propertyDefinitionGet,
	Long:  usagePropertyDefinitionGet,
}

func setupPropertyDefinitionGetFlags() {
	propertyDefinitionGetCommand.Flags().StringArrayVarP(
		&cliFilters,
		"filter", "f",
		nil,
		usagePropertyDefinitionFilterOption,
	)
}

func init() {
	setupPropertyDefinitionGetFlags()
}

func propertyDefinitionGet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)

	session := getSession()

	var filter *pb.PropertyDefinitionFilter

	if len(args) > 1 {
		fmt.Fprintf(
			os.Stderr,
			"Error: either specify a <UUID> argument or a single "+
				"--filter option string\n",
		)
		cmd.Help()
		os.Exit(1)
	}

	if len(args) == 1 {
		filter = &pb.PropertyDefinitionFilter{
			Uuid: args[0],
		}
	} else {
		filters := buildPropertyDefinitionFilters()
		if len(filters) != 1 {
			fmt.Fprintf(
				os.Stderr,
				"Error: either specify a <UUID> argument or a single "+
					"--filter option string\n",
			)
			cmd.Help()
			os.Exit(1)
		}
		filter = filters[0]
	}

	req := &pb.PropertyDefinitionGetRequest{
		Session: session,
		Filter:  filter,
	}
	obj, err := client.PropertyDefinitionGet(context.Background(), req)
	exitIfError(err)
	printPropertyDefinition(obj)
}

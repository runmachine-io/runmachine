package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

const (
	usageObjectGet = `Show information for a single object

There are two call signatures to this command.

The first is to specify a single CLI argument that should be the UUID of the
object you wish to show:

  runm object get 4f4f54c9bfb44cce9a02d4daf6f79ea3

The second is to specify a --filter option string that returns a single
object.

  runm object get --filter "type=runm.image name=fedora23"

Specifying a --filter option string that returns more than one obect will
result in a MultipleRecordsFound error.

The --filter option string will ignored when also supplying a <UUID> argument.
`
)

var objectGetCommand = &cobra.Command{
	Use:   "get [<UUID>]",
	Short: "Show information for a single object",
	Run:   objectGet,
	Long:  usageObjectGet,
}

func setupObjectGetFlags() {
	objectGetCommand.Flags().StringArrayVarP(
		&cliFilters,
		"filter", "f",
		nil,
		usageObjectFilterOption,
	)
}

func init() {
	setupObjectGetFlags()
}

func objectGet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)

	session := getSession()

	var filter *pb.ObjectFilter

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
		filter = &pb.ObjectFilter{
			Uuid: args[0],
		}
	} else {
		filters := buildObjectFilters()
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
	obj, err := client.ObjectGet(
		context.Background(),
		&pb.ObjectGetRequest{
			Session: session,
			Filter:  filter,
		},
	)
	exitIfError(err)
	printObject(obj)
}

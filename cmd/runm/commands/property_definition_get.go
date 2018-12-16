package commands

import (
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

const (
	usagePropDefGetPartitionOpt = `optional partition filter.

If not set, defaults to the partition used in the user's session.
`
)

var (
	// partition override. if empty, we use the session's partition
	cliPropDefGetPartition string
)

var propertyDefinitionGetCommand = &cobra.Command{
	Use:   "get <object_type> <key>",
	Short: "Show information for a single property definition",
	Args:  cobra.ExactArgs(2),
	Run:   propertyDefinitionGet,
}

func setupPropertyDefinitionGetFlags() {
	propertyDefinitionGetCommand.Flags().StringVarP(
		&cliPropDefGetPartition,
		"partition", "", "",
		usagePropDefGetPartitionOpt,
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

	filter := &pb.PropertyDefinitionFilter{
		Type: &pb.ObjectTypeFilter{
			Search:    args[0],
			UsePrefix: false,
		},
		Search:    args[1],
		UsePrefix: false,
	}
	if cliPropDefGetPartition != "" {
		filter.Partition = &pb.PartitionFilter{
			Search:    cliPropDefGetPartition,
			UsePrefix: false,
		}
	}

	req := &pb.PropertyDefinitionGetRequest{
		Session: session,
		Filter:  filter,
	}
	obj, err := client.PropertyDefinitionGet(context.Background(), req)
	exitIfError(err)
	printPropertyDefinition(obj)
}

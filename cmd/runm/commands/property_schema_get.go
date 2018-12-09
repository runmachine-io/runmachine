package commands

import (
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

const (
	usagePropSchemaGetPartitionOpt = `optional partition filter.

If not set, defaults to the partition used in the user's session.
`
)

var (
	// partition override. if empty, we use the session's partition
	propSchemaGetPartition string
)

var propertySchemaGetCommand = &cobra.Command{
	Use:   "get <object_type> <key>",
	Short: "Show information for a single property schema",
	Args:  cobra.ExactArgs(2),
	Run:   propertySchemaGet,
}

func setupPropertySchemaGetFlags() {
	propertySchemaGetCommand.Flags().StringVarP(
		&propSchemaGetPartition,
		"partition", "", "",
		usagePropSchemaGetPartitionOpt,
	)
}

func init() {
	setupPropertySchemaGetFlags()
}

func propertySchemaGet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)

	session := getSession()

	filter := &pb.PropertySchemaFilter{
		Type: &pb.ObjectTypeFilter{
			Search:    args[0],
			UsePrefix: false,
		},
		Search:    args[1],
		UsePrefix: false,
	}
	if propSchemaGetPartition != "" {
		filter.Partition = &pb.PartitionFilter{
			Search:    propSchemaGetPartition,
			UsePrefix: false,
		}
	}

	req := &pb.PropertySchemaGetRequest{
		Session: session,
		Filter:  filter,
	}
	obj, err := client.PropertySchemaGet(context.Background(), req)
	exitIfError(err)
	printPropertySchema(obj)
}

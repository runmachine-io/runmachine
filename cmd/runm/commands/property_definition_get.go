package commands

import (
	"fmt"
	"os"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

const (
	usagePropDefGetUuidOpt       = `search by the property definition's UUID`
	usagePropDefGetObjectTypeOpt = `search by the property definition's object type

NOTE: Using this option will require specifying the property definition's key
using the --key CLI option.
`
	usagePropDefGetKeyOpt = `search by the property definition's key

NOTE: Using this option will require specifying the property definition's
object type using the --object-type CLI option.
`
	usagePropDefGetPartitionOpt = `optional partition filter.

If not set, defaults to the partition used in the user's session.
`
)

var (
	// Search by UUID
	cliPropDefGetUuid string
	// Search by object type (and key)
	cliPropDefGetObjectType string
	cliPropDefGetKey        string
	// partition override. if empty, we use the session's partition
	cliPropDefGetPartition string
)

var propertyDefinitionGetCommand = &cobra.Command{
	Use:   "get",
	Short: "Show information for a single property definition",
	Run:   propertyDefinitionGet,
}

func setupPropertyDefinitionGetFlags() {
	propertyDefinitionGetCommand.Flags().StringVarP(
		&cliPropDefGetUuid,
		"uuid", "", "",
		usagePropDefGetUuidOpt,
	)
	propertyDefinitionGetCommand.Flags().StringVarP(
		&cliPropDefGetObjectType,
		"object-type", "", "",
		usagePropDefGetObjectTypeOpt,
	)
	propertyDefinitionGetCommand.Flags().StringVarP(
		&cliPropDefGetKey,
		"key", "", "",
		usagePropDefGetKeyOpt,
	)
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

	filter := &pb.PropertyDefinitionFilter{}

	if cliPropDefGetUuid != "" {
		filter.Uuid = cliPropDefGetUuid
	} else {
		if cliPropDefGetObjectType == "" || cliPropDefGetKey == "" {
			fmt.Fprintf(
				os.Stderr,
				"Error: either specify --uuid or specify *BOTH* "+
					"--object-type <TYPE> and --key <KEY>\n",
			)
			os.Exit(1)
		}
		filter.Type = &pb.ObjectTypeFilter{
			Search: cliPropDefGetObjectType,
		}
		filter.Key = cliPropDefGetKey
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

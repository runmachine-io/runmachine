package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	"github.com/runmachine-io/runmachine/pkg/util"
	pb "github.com/runmachine-io/runmachine/proto"
)

const (
	usageObjectGetType = `specify the object type.

Required when <search> is not a UUID.
`
)

var (
	// CLI-provided --type value
	cliObjectGetType string
)

var objectGetCommand = &cobra.Command{
	Use:   "get <search>",
	Short: "Show information for a single object",
	Args:  cobra.ExactArgs(1),
	Run:   objectGet,
}

func setupObjectGetFlags() {
	objectGetCommand.Flags().StringVarP(
		&cliObjectGetType,
		"type", "t", "",
		usageObjectGetType,
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
	uuidOrName := args[0]
	search := &pb.ObjectFilter{
		Search:    uuidOrName,
		UsePrefix: false,
	}

	if !util.IsUuidLike(uuidOrName) {
		if cliObjectGetType == "" {
			fmt.Fprintf(os.Stderr, "Error: --type required when <search> is not a UUID\n")
			os.Exit(1)
		}
		search.Type = &pb.ObjectTypeFilter{
			Search:    cliObjectGetType,
			UsePrefix: false,
		}
	}
	obj, err := client.ObjectGet(
		context.Background(),
		&pb.ObjectGetRequest{
			Session: session,
			Search:  search,
		},
	)
	exitIfError(err)
	fmt.Printf("Partition:   %s\n", obj.Partition)
	fmt.Printf("Type:        %s\n", obj.Type)
	fmt.Printf("UUID:        %s\n", obj.Uuid)
	fmt.Printf("Name:        %s\n", obj.Name)
	if obj.Project != "" {
		fmt.Printf("Project:     %s\n", obj.Project)
	}
	if obj.Properties != nil {
		fmt.Printf("Properties:\n")
		for _, prop := range obj.Properties {
			fmt.Printf("   %s=%s\n", prop.Key, prop.Value)
		}
	}
	if obj.Tags != nil {
		tags := strings.Join(obj.Tags, ",")
		fmt.Printf("Tags:       %s\n", tags)
	}
}

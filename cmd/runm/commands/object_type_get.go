package commands

import (
	"fmt"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/spf13/cobra"
)

var objectTypeGetCommand = &cobra.Command{
	Use:   "get <code>",
	Short: "Show information for a single object type",
	Args:  cobra.ExactArgs(1),
	Run:   objectTypeGet,
}

func objectTypeGet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)

	session := getSession()

	req := &pb.ObjectTypeGetRequest{
		Session: session,
		Filter: &pb.ObjectTypeFilter{
			Search:    args[0],
			UsePrefix: false,
		},
	}
	obj, err := client.ObjectTypeGet(context.Background(), req)
	exitIfError(err)
	fmt.Printf("Code:        %s\n", obj.Code)
	fmt.Printf("Scope:       %s\n", obj.Scope.String())
	fmt.Printf("Description: %s\n", obj.Description)
}

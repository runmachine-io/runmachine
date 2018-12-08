package commands

import (
	"fmt"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var objectCreateCommand = &cobra.Command{
	Use:   "create",
	Short: "Create a new object",
	Run:   objectCreate,
}

func setupObjectCreateFlags() {
	objectCreateCommand.Flags().StringVarP(
		&objectDocPath,
		"file", "f",
		"",
		"optional filepath to YAML document to send.",
	)
}

func init() {
	setupObjectCreateFlags()
}

func objectCreate(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.ObjectSetRequest{
		Session: getSession(),
		Format:  pb.PayloadFormat_YAML,
		Payload: readInputDocumentOrExit(),
	}

	resp, err := client.ObjectSet(context.Background(), req)
	exitIfError(err)
	obj := resp.Object
	if !quiet {
		fmt.Printf("UUID:        %s\n", obj.Uuid)
		fmt.Printf("Type:        %s\n", obj.Type)
		fmt.Printf("Partition:   %s\n", obj.Partition)
		fmt.Printf("Name:        %s\n", obj.Name)
		fmt.Printf("Project:     %s\n", obj.Project)
	} else {
		fmt.Println(obj.Uuid)
	}
}

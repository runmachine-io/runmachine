package commands

import (
	"fmt"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var objectSetCommand = &cobra.Command{
	Use:   "set",
	Short: "Create or update an object",
	Run:   objectSet,
}

func setupObjectSetFlags() {
	objectSetCommand.Flags().StringVarP(
		&cliObjectDocPath,
		"file", "f",
		"",
		"optional filepath to YAML document to send.",
	)
}

func init() {
	setupObjectSetFlags()
}

func objectSet(cmd *cobra.Command, args []string) {
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
		fmt.Printf("ok\n")
		if verbose {
			printObject(obj)
		}
	}
}

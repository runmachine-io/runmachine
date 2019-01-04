package commands

import (
	"fmt"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/spf13/cobra"
)

var partitionCreateCommand = &cobra.Command{
	Use:   "create",
	Short: "Create a partition",
	Run:   partitionCreate,
}

func setupPartitionCreateFlags() {
	partitionCreateCommand.Flags().StringVarP(
		&cliObjectDocPath,
		"file", "f",
		"",
		"optional filepath to YAML document to send.",
	)
}

func init() {
	setupPartitionCreateFlags()
}

func partitionCreate(cmd *cobra.Command, args []string) {
	conn := apiConnect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)
	req := &pb.CreateRequest{
		Session: apiGetSession(),
		Format:  pb.PayloadFormat_YAML,
		Payload: readInputDocumentOrExit(),
	}

	resp, err := client.PartitionCreate(context.Background(), req)
	exitIfError(err)
	obj := resp.Partition
	if !quiet {
		fmt.Printf("ok\n")
		if verbose {
			printPartition(obj)
		}
	}
}

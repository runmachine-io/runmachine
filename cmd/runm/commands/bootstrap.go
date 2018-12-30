package commands

import (
	"fmt"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
	"github.com/spf13/cobra"
)

var (
	// Optional UUID for the partition to create
	bootstrapPartitionUuid string
)

var bootstrapCommand = &cobra.Command{
	Use:   "bootstrap <token> <partition_name>",
	Short: "Bootstrap using a one-time-use token. Creates a new partition.",
	Args:  cobra.ExactArgs(2),
	Run:   bootstrap,
}

func setupBootstrapFlags() {
	bootstrapCommand.Flags().StringVarP(
		&bootstrapPartitionUuid,
		"partition-uuid", "",
		"",
		"Optional UUID of the partition to create.",
	)
}

func init() {
	setupBootstrapFlags()
}

func bootstrap(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.BootstrapRequest{
		BootstrapToken: args[0],
		PartitionName:  args[1],
	}

	if bootstrapPartitionUuid != "" {
		req.PartitionUuid = &pb.StringValue{Value: bootstrapPartitionUuid}
	}

	resp, err := client.Bootstrap(context.Background(), req)
	exitIfError(err)
	obj := resp.Partition
	fmt.Println(obj.Uuid)
}

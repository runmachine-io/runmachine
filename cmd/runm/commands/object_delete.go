package commands

import (
	"fmt"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var objectDeleteCommand = &cobra.Command{
	Use:   "delete",
	Short: "Delete objects matching one or more filters",
	Run:   objectDelete,
}

func setupObjectDeleteFlags() {
	objectDeleteCommand.Flags().StringArrayVarP(
		&cliFilters,
		"filter", "f",
		nil,
		usageObjectFilterOption,
	)
}

func init() {
	setupObjectDeleteFlags()
}

func objectDelete(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.ObjectDeleteRequest{
		Session: getSession(),
		Any:     buildObjectFilters(),
	}
	resp, err := client.ObjectDelete(context.Background(), req)
	if s, ok := status.FromError(err); ok {
		if s.Code() != codes.OK {
			fmt.Fprintf(os.Stderr, "Error: %s\n", s.Message())
			if resp != nil && len(resp.Errors) > 0 {
				fmt.Fprintf(os.Stderr, "Details:\n")
				for x, errText := range resp.Errors {
					fmt.Fprintf(os.Stderr, "%d: %s\n", x, errText)
				}
			}
			os.Exit(int(s.Code()))
		}
	}
	if !quiet {
		if verbose {
			fmt.Fprintf(os.Stdout, "deleted %d object(s)\n", resp.NumDeleted)
		} else {
			fmt.Fprintf(os.Stdout, "ok\n")
		}
	}
}

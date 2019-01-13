package commands

import (
	"fmt"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/spf13/cobra"
)

const (
	providerDeleteUsage = `runm provider delete may be called in two ways:

The first way is to specify provider identifiers (provider name or UUID) as
arguments. For example, to delete a provider with the UUID
873109dc73e343e19e7bcc51a3ef4db5, you would call:

  runm provider delete 873109dc73e343e19e7bcc51a3ef4db5

Multiple identifiers may be passed to delete multiple providers. For example,
to delete providers with the names "compute1" and "compute2", you would call:

  runm provider delete compute1 compute2

The second way is to specify a "--filter <expression>" CLI option. All
providers matching the filter expression will be deleted. For example, if you
wanted to delete all providers in a partition called "part42", you would call:

  runm provider delete --filter "partition=part42"
`
)

var providerDeleteCommand = &cobra.Command{
	Use:   "delete [<id> ...]",
	Short: "Delete providers matching one or more filters",
	Run:   providerDelete,
	Long:  providerDeleteUsage,
}

func setupProviderDeleteFlags() {
	providerDeleteCommand.Flags().StringArrayVarP(
		&cliFilters,
		"filter", "f",
		nil,
		usageProviderFilterOption,
	)
}

func init() {
	setupProviderDeleteFlags()
}

func providerDelete(cmd *cobra.Command, args []string) {
	conn := apiConnect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)
	req := &pb.ProviderDeleteRequest{
		Session: apiGetSession(),
	}

	// See providerDeleteUsage above that describes the two calling signatures
	// for `runm provider delete` and why we use either the --filter
	// expressions *or* each CLI argument passed as a separate lookup
	if len(args) == 0 {
		req.Any = buildProviderFilters()
	} else {
		// We treat each argument as a Name-or-UUID filter
		filters := make([]*pb.ProviderFilter, len(args))
		for x, arg := range args {
			filters[x] = &pb.ProviderFilter{
				PrimaryFilter: &pb.SearchFilter{
					Search:    arg,
					UsePrefix: false,
				},
			}
		}
		req.Any = filters
	}

	resp, err := client.ProviderDelete(context.Background(), req)
	if s, ok := status.FromError(err); ok {
		if s.Code() != codes.OK {
			fmt.Fprintf(os.Stderr, "Error: %s\n", s.Message())
			os.Exit(int(s.Code()))
		}
	}
	if !quiet {
		if verbose {
			fmt.Fprintf(os.Stdout, "deleted %d provider(s)\n", resp.NumDeleted)
		} else {
			fmt.Fprintf(os.Stdout, "ok\n")
		}
	}
}

package commands

import (
	"fmt"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/pkg/api/proto"
	"github.com/spf13/cobra"
)

var providerCreateCommand = &cobra.Command{
	Use:   "create",
	Short: "Create a provider",
	Run:   providerCreate,
}

func setupProviderCreateFlags() {
	providerCreateCommand.Flags().StringVarP(
		&cliObjectDocPath,
		"file", "f",
		"",
		"optional filepath to YAML document to send.",
	)
}

func init() {
	setupProviderCreateFlags()
}

func providerCreate(cmd *cobra.Command, args []string) {
	conn := apiConnect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)
	req := &pb.ProviderCreateRequest{
		Session: apiGetSession(),
		Format:  pb.PayloadFormat_YAML,
		Payload: readInputDocumentOrExit(),
	}

	resp, err := client.ProviderCreate(context.Background(), req)
	exitIfError(err)
	obj := resp.Provider
	if !quiet {
		fmt.Printf("ok\n")
		if verbose {
			printProvider(obj)
		}
	}
}

package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
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
	conn := connect()
	defer conn.Close()

	client := pb.NewRunmAPIClient(conn)
	req := &pb.CreateRequest{
		Session: getSession(),
		Format:  pb.PayloadFormat_YAML,
		Payload: readInputDocumentOrExit(),
	}

	resp, err := client.ProviderCreate(context.Background(), req)
	exitIfError(err)
	obj := resp.Provider
	if !quiet {
		if verbose {
			printProvider(obj)
		} else {
			fmt.Printf("%s\n", obj.Uuid)
		}
	}
}

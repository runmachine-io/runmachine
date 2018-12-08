package commands

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/net/context"

	pb "github.com/runmachine-io/runmachine/proto"
	"github.com/spf13/cobra"
)

var (
	// optional filepath to read the object file containing the schema from
	objectDocPath string
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

	// TODO(jaypipes): Move the below generic file-reading into a common helper
	// function since this is going to be a common pattern for
	// creating/updating objects.
	var b []byte
	if objectDocPath == "" {
		// User did not specify -f therefore we expect to read the YAML
		// document from stdin
		scanner := bufio.NewScanner(os.Stdin)
		buf := make([]byte, 0)
		for scanner.Scan() {
			buf = append(buf, scanner.Bytes()...)
		}
		b = buf
	} else {
		if buf, err := ioutil.ReadFile(objectDocPath); err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		} else {
			b = buf
		}
	}

	if len(b) == 0 {
		fmt.Println("Error: expected to receive object YAML in STDIN")
		os.Exit(1)
	}

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.ObjectSetRequest{
		Session: getSession(),
		Format:  pb.PayloadFormat_YAML,
		Payload: b,
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

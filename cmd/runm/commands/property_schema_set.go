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
	// YAML document we will read stdin or the -f option's value
	propSchemaDoc string
	// optional filepath to read the object file containing the schema from
	propSchemaDocPath string
)

var propertySchemaSetCommand = &cobra.Command{
	Use:   "set",
	Short: "Create or update a property schema",
	Run:   propertySchemaSet,
}

func setupPropertySchemaSetFlags() {
	propertySchemaSetCommand.Flags().StringVarP(
		&propSchemaDocPath,
		"file", "f",
		"",
		"optional filepath to property schema document to send.",
	)
}

func init() {
	setupPropertySchemaSetFlags()
}

// propertySchemaProcessStdin reads the supplied buffer which contains a YAML
// document describing the property schema to create or update, and returns a
// pointer to PropertySchemaSetFields protobuffer message containing the fields
// to set on the new (or changed) object.
func getPropertySchemaFromFile(b []byte) (*pb.PropertySchema, error) {
	return &pb.PropertySchema{}, nil
}

func propertySchemaSet(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

	var b []byte
	if propSchemaDocPath == "" {
		// User did not specify -f therefore we expect to read the property
		// schema YAML document from stdin
		scanner := bufio.NewScanner(os.Stdin)
		buf := make([]byte, 0)
		for scanner.Scan() {
			buf = append(buf, scanner.Bytes()...)
		}
		b = buf
	} else {
		if buf, err := ioutil.ReadFile(propSchemaDocPath); err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		} else {
			b = buf
		}
	}

	if len(b) == 0 {
		fmt.Println("Error: expected to receive property schema YAML in STDIN")
		os.Exit(1)
	}

	obj, err := getPropertySchemaFromFile(b)
	if err != nil {
		fmt.Printf("Error: failed to parse property schema YAML document: %s\n", err)
		os.Exit(1)
	}

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.PropertySchemaSetRequest{
		Session:        getSession(),
		PropertySchema: obj,
	}

	resp, err := client.PropertySchemaSet(context.Background(), req)
	exitIfError(err)
	obj = resp.PropertySchema
	if !quiet {
		fmt.Printf("Successfully created property schema\n")
		if verbose {
			fmt.Printf("Partition:    %s\n", obj.Partition)
			fmt.Printf("Object type:  %s\n", obj.ObjectType)
			fmt.Printf("Key:          %s\n", obj.Key)
			fmt.Printf("Schema:\n%s\n", obj.Schema)
		}
	}
}

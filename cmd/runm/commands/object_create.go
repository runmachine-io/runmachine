package commands

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/net/context"
	yaml "gopkg.in/yaml.v2"

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
		"optional filepath to object document to read.",
	)
}

func init() {
	setupObjectCreateFlags()
}

// The YAML document will be parsed into this struct, which is similar to an
// Object protobuffer message
type objDoc struct {
	Partition string `yaml:"partition"`
	Type      string `yaml:"type"`
	Project   string `yaml:"project"`
	Name      string `yaml:"name"`
	// TODO(jaypipes): Handle properties and tags...
}

// getObjectFromBytes reads the supplied buffer which contains a YAML document
// describing the object to create or update, and returns a pointer to an
// Object protobuffer message containing the fields to set on the new (or
// changed) object.
func getObjectFromBytes(b []byte) (*pb.Object, error) {
	od := &objDoc{}
	if err := yaml.Unmarshal(b, od); err != nil {
		return nil, err
	}
	return &pb.Object{
		// The server actually will translate partition names to UUIDs...
		Partition:  od.Partition,
		ObjectType: od.Type,
		Project:    od.Project,
		Name:       od.Name,
	}, nil
}

func objectCreate(cmd *cobra.Command, args []string) {
	conn := connect()
	defer conn.Close()

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

	obj, err := getObjectFromBytes(b)
	if err != nil {
		fmt.Printf("Error: failed to parse object YAML document: %s\n", err)
		os.Exit(1)
	}

	client := pb.NewRunmMetadataClient(conn)
	req := &pb.ObjectSetRequest{
		Session: getSession(),
		After:   obj,
	}

	resp, err := client.ObjectSet(context.Background(), req)
	exitIfError(err)
	obj = resp.Object
	if !quiet {
		fmt.Printf("UUID:        %s\n", obj.Uuid)
		fmt.Printf("Type:        %s\n", obj.ObjectType)
		fmt.Printf("Partition:   %s\n", obj.Partition)
		fmt.Printf("Name:        %s\n", obj.Name)
		fmt.Printf("Project:     %s\n", obj.Project)
	} else {
		fmt.Println(obj.Uuid)
	}
}

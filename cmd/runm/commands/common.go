package commands

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apipb "github.com/runmachine-io/runmachine/pkg/api/proto"
	pb "github.com/runmachine-io/runmachine/pkg/metadata/proto"
)

const (
	permissionsHelpExtended = `

    NOTE: To find out what permissions may be applied to a role, use
          the runm permissions command.
`
)

const (
	rolesHelpExtended = `

    NOTE: To find out what roles a user may be added to, use
          the runm role list command.
`
)

const (
	errUnsetUser = `Error: unable to find the authenticating user.

Please set the RUNM_USER environment variable or supply a value
for the --user CLI option.
`
	errConnect = `Error: unable to connect to the runm-metadata server.

Please check the RUNM_HOST and RUNM_PORT environment
variables or --host and --port  CLI options.
`
	errForbidden = `Error: you are not authorized to perform that action.`
)

const (
	msgNoRecords = "No records found matching search criteria."
)

// Some commonly-used CLI options
const (
	defaultListLimit = 50
	defaultListSort  = "uuid:asc"
)

var (
	listLimit  int
	listMarker string
	listSort   string
	// filepath to read a document to send to the server for create/update operations
	cliObjectDocPath string
	// CLI-provided set of --filter options
	cliFilters = []string{}
)

func exitIfConnectErr(err error) {
	if err != nil {
		fmt.Println(errConnect)
		os.Exit(1)
	}
}

// Writes a generic error to output and exits if supplied error is an error
func exitIfError(err error) {
	if s, ok := status.FromError(err); ok {
		if s.Code() != codes.OK {
			fmt.Fprintf(os.Stderr, "Error: %s\n", s.Message())
			os.Exit(int(s.Code()))
		}
	}
}

func exitNoRecords() {
	if !quiet {
		fmt.Println(msgNoRecords)
	}
	os.Exit(0)
}

func readInputDocumentOrExit() []byte {
	var b []byte
	if cliObjectDocPath == "" {
		// User did not specify -f therefore we expect to read the YAML
		// document from stdin
		scanner := bufio.NewScanner(os.Stdin)
		buf := make([]byte, 0)
		for scanner.Scan() {
			buf = append(buf, scanner.Bytes()...)
		}
		b = buf
	} else {
		if buf, err := ioutil.ReadFile(cliObjectDocPath); err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		} else {
			b = buf
		}
	}

	if len(b) == 0 {
		fmt.Println("Error: expected to receive YAML document in STDIN")
		os.Exit(1)
	}
	return b
}

// getSession constructs a Session protobuffer message by looking for
// partition, user and project information in a variety of configuration file,
// CLI argument and environs variable locations.
func getSession() *pb.Session {
	user := authUser
	project := authProject
	partition := authPartition
	if user == "" || project == "" || partition == "" {
		// TODO(jaypipes): Load a YAML configuration file where we might be
		// able to find missing user/project/partition information
	}
	return &pb.Session{
		User:      user,
		Project:   project,
		Partition: partition,
	}
}
func apiGetSession() *apipb.Session {
	sess := getSession()
	return &apipb.Session{
		User:      sess.User,
		Project:   sess.Project,
		Partition: sess.Partition,
	}
}

func connect() *grpc.ClientConn {
	var opts []grpc.DialOption
	// TODO(jaypipes): Don't hardcode this to WithInsecure
	opts = append(opts, grpc.WithInsecure())
	addr := fmt.Sprintf("%s:%d", connectHost, connectPort)
	printIf(verbose, "connecting to runm services at %s\n", addr)
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		fmt.Println(errConnect)
		os.Exit(1)
		return nil
	}
	return conn
}

func apiConnect() *grpc.ClientConn {
	var opts []grpc.DialOption
	// TODO(jaypipes): Don't hardcode this to WithInsecure
	opts = append(opts, grpc.WithInsecure())
	addr := fmt.Sprintf("%s:%d", apiConnectHost, apiConnectPort)
	printIf(verbose, "connecting to runm-api service at %s\n", addr)
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		fmt.Println(errConnect)
		os.Exit(1)
		return nil
	}
	return conn
}

func printIf(b bool, msg string, args ...interface{}) {
	if b {
		fmt.Printf(msg, args...)
	}
}

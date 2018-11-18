package commands

import (
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/runmachine-io/runmachine/proto"
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
			fmt.Printf("Error: %s\n", s.Message())
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

func printIf(b bool, msg string, args ...interface{}) {
	if b {
		fmt.Printf(msg, args...)
	}
}

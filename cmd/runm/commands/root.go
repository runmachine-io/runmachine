package commands

import (
	"io/ioutil"
	"log"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/grpclog"

	"github.com/jaypipes/envutil"
)

type Logger grpclog.Logger

const (
	quietHelpExtended = `

    NOTE: For commands that create, modify or delete an object, the
          --quiet flag triggers the outputting of the newly-created
          object's identifier as the only output from the command.
          For commands that update an object's state, this will quiet
          all output, meaning the user will need to query the result
          code from the runm program in order to determine success.
`
)

const (
	defaultConnectHost = "localhost"
	defaultConnectPort = 10000
)

var (
	quiet       bool
	verbose     bool
	connectHost string
	connectPort int
	authUser    string
	clientLog   Logger
)

var RootCommand = &cobra.Command{
	Use:   "runm",
	Short: "runm - the runmachine CLI tool.",
	Long:  "Manipulate a runmachine system.",
}

func addConnectFlags() {
	RootCommand.PersistentFlags().BoolVarP(
		&quiet,
		"quiet", "q",
		false,
		"Show minimal output."+quietHelpExtended,
	)
	RootCommand.PersistentFlags().BoolVarP(
		&verbose,
		"verbose", "v",
		false,
		"Show more output.",
	)
	RootCommand.PersistentFlags().StringVarP(
		&connectHost,
		"host", "",
		envutil.WithDefault(
			"RUNM_HOST",
			defaultConnectHost,
		),
		"The host where the runmachine API can be found.",
	)
	RootCommand.PersistentFlags().IntVarP(
		&connectPort,
		"port", "",
		envutil.WithDefaultInt(
			"RUNM_PORT",
			defaultConnectPort,
		),
		"The port where the runmachine API can be found.",
	)
	RootCommand.PersistentFlags().StringVarP(
		&authUser,
		"user", "",
		envutil.WithDefault(
			"RUNM_USER",
			"",
		),
		"UUID, email or \"slug\" of the user to execute commands with.",
	)
}

func init() {
	addConnectFlags()

	RootCommand.AddCommand(helpEnvCommand)
	RootCommand.AddCommand(propertySchemaCommand)
	RootCommand.SilenceUsage = true

	clientLog = log.New(ioutil.Discard, "", 0)
	grpclog.SetLogger(clientLog)
}

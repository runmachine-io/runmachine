package commands

import (
	"io/ioutil"
	"log"

	"github.com/jaypipes/envutil"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/grpclog"
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
	defaultConnectHost    = "localhost"
	defaultConnectPort    = 10000
	defaultAPIConnectPort = 10002
)

var (
	quiet          bool
	verbose        bool
	connectHost    string
	connectPort    int
	apiConnectPort int
	apiConnectHost string
	authPartition  string
	authUser       string
	authProject    string
	clientLog      Logger
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
		&apiConnectHost,
		"api-host", "",
		envutil.WithDefault(
			"RUNM_API_HOST",
			defaultConnectHost,
		),
		"The host where the runmachine API can be found.",
	)
	RootCommand.PersistentFlags().IntVarP(
		&apiConnectPort,
		"api-port", "",
		envutil.WithDefaultInt(
			"RUNM_API_PORT",
			defaultAPIConnectPort,
		),
		"The port where the runmachine API can be found.",
	)
	RootCommand.PersistentFlags().StringVarP(
		&authPartition,
		"partition", "",
		envutil.WithDefault(
			"RUNM_PARTITION",
			"",
		),
		"UUID or name of the partition to execute commands against.",
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
	RootCommand.PersistentFlags().StringVarP(
		&authProject,
		"project", "",
		envutil.WithDefault(
			"RUNM_PROJECT",
			"",
		),
		"UUID or name of the project to execute commands under.",
	)
}

func init() {
	addConnectFlags()

	RootCommand.AddCommand(helpEnvCommand)
	RootCommand.AddCommand(bootstrapCommand)
	RootCommand.AddCommand(objectCommand)
	RootCommand.AddCommand(propertyDefinitionCommand)
	RootCommand.AddCommand(partitionCommand)
	RootCommand.AddCommand(providerCommand)
	RootCommand.AddCommand(providerTypeCommand)
	RootCommand.SilenceUsage = true

	clientLog = log.New(ioutil.Discard, "", 0)
	grpclog.SetLogger(clientLog)
}

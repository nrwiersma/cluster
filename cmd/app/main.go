package main

import (
	"log"
	"os"

	"github.com/hamba/cmd"
	"gopkg.in/urfave/cli.v2"
)

import _ "github.com/joho/godotenv/autoload"

const (
	flagID              = "id"
	flagName            = "name"
	flagDataDir         = "data-dir"
	flagSerfAddr        = "serf-addr"
	flagEncryptKey      = "encrypt"
	flagRPCAddr         = "rpc-addr"
	flagBootstrap       = "bootstrap"
	flagBootstrapExpect = "bootstrap-expect"
	flagJoin            = "join"
)

var version = "¯\\_(ツ)_/¯"

var commands = []*cli.Command{
	{
		Name:  "agent",
		Usage: "Run the cluster agent",
		Flags: cmd.Flags{
			&cli.StringFlag{
				Name:    flagID,
				Usage:   "The agent id.",
				EnvVars: []string{"AGENT_ID"},
			},
			&cli.StringFlag{
				Name:    flagName,
				Usage:   "The node name.",
				EnvVars: []string{"AGENT_NAME"},
			},
			&cli.StringFlag{
				Name:    flagDataDir,
				Usage:   "The path under which to store log files.",
				Value:   "/tmp/cluster",
				EnvVars: []string{"AGENT_DATA_DIR"},
			},
			&cli.StringFlag{
				Name:    flagSerfAddr,
				Usage:   "The address for Serf to bind on.",
				Value:   "0.0.0.0:8301",
				EnvVars: []string{"AGENT_SERF_ADDR"},
			},
			&cli.StringFlag{
				Name:    flagEncryptKey,
				Usage:   "The encryption key to secure Serf.",
				EnvVars: []string{"AGENT_ENCRYPTION_KEY"},
			},
			&cli.StringFlag{
				Name:    flagRPCAddr,
				Usage:   "The address for the RPC to bind and advertise on.",
				Value:   "127.0.0.1:8300",
				EnvVars: []string{"AGENT_RPC_ADDR"},
			},
			&cli.BoolFlag{
				Name:    flagBootstrap,
				Usage:   "Initial cluster bootstrapping.",
				EnvVars: []string{"AGENT_BOOTSTRAP"},
			},
			&cli.IntFlag{
				Name:    flagBootstrapExpect,
				Usage:   "The number of expected agents in the cluster.",
				EnvVars: []string{"AGENT_BOOTSTRAP_EXPECT"},
			},
			&cli.StringSliceFlag{
				Name:    flagJoin,
				Usage:   "The serf addresses of agents to join at start time.",
				Value:   nil,
				EnvVars: []string{"AGENT_JOIN"},
			},
		}.Merge(cmd.CommonFlags),
		Action: runAgent,
	},
	{
		Name:   "keygen",
		Usage:  "Generate an encryption key",
		Action: runKeyGen,
	},
}

func newApp() *cli.App {
	return &cli.App{
		Name:     "app",
		Version:  version,
		Commands: commands,
	}
}

func main() {
	app := newApp()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

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
	flagRaftAddr        = "raft-addr"
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
			&cli.IntFlag{
				Name:    flagID,
				Usage:   "The agent id.",
				Value:   0,
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
				Name:    flagRaftAddr,
				Usage:   "The address for Raft to bind and advertise on.",
				Value:   "127.0.0.1:8300",
				EnvVars: []string{"AGENT_RAFT_ADDR"},
			},
			&cli.BoolFlag{
				Name:    flagBootstrap,
				Usage:   "Initial cluster bootstrapping.",
				EnvVars: []string{"AGENT_BOOTSTRAP"},
			},
			&cli.IntFlag{
				Name:    flagBootstrapExpect,
				Usage:   "The number of expected agents in the cluster.",
				EnvVars: []string{"AGENT_EXPECT"},
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

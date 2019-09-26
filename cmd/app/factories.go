package main

import (
	"net"
	"strconv"

	"github.com/hamba/cmd"
	"github.com/nrwiersma/cluster"
	clus "github.com/nrwiersma/cluster/cluster"
)

// Application =============================

func newApplication(c *cmd.Context) (*cluster.Application, error) {
	app := cluster.NewApplication(
		c.Logger(),
		c.Statter(),
	)

	// Setup your application here

	return app, nil
}

// Agent ===================================

func newAgent(c *cmd.Context) (*clus.Agent, error) {
	cfg := clus.NewConfig()
	cfg.DataDir = c.String(flagDataDir)
	cfg.EncryptKey = c.String(flagEncryptKey)
	cfg.RPCAddr = c.String(flagRPCAddr)
	cfg.Bootstrap = c.Bool(flagBootstrap)
	cfg.BootstrapExpect = c.Int(flagBootstrapExpect)
	cfg.Logger = c.Logger()

	if id := c.String(flagID); id != "" {
		cfg.ID = id
	}

	if name := c.String(flagName); name != "" {
		cfg.Name = name
	}

	// Setup the serf addr
	bindIP, bindPort, err := net.SplitHostPort(c.String(flagSerfAddr))
	if err != nil {
		return nil, err
	}
	cfg.SerfConfig.MemberlistConfig.BindAddr = bindIP
	cfg.SerfConfig.MemberlistConfig.BindPort, err = strconv.Atoi(bindPort)
	if err != nil {
		return nil, err
	}

	return clus.New(cfg)
}

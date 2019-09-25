package main

import (
	"fmt"
	"math/rand"
	"net"
	"path/filepath"
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
	id := c.Int(flagID)
	if id == 0 {
		id = rand.Int()
		c.Logger().Info(fmt.Sprintf("agent: ID not selected; using %d", id))
	}

	// TODO: remove this, temp
	dataDir := filepath.Join(c.String(flagDataDir), strconv.Itoa(id))

	cfg := clus.NewConfig()
	cfg.ID = int32(id)
	cfg.DataDir = dataDir
	cfg.EncryptKey = c.String(flagEncryptKey)
	cfg.RPCAddr = c.String(flagRaftAddr)
	cfg.Bootstrap = c.Bool(flagBootstrap)
	cfg.BootstrapExpect = c.Int(flagBootstrapExpect)
	cfg.Logger = c.Logger()

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

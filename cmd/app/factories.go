package main

import (
	"fmt"
	"net"
	"strconv"

	"github.com/hamba/cmd"
	"github.com/nrwiersma/cluster"
	clus "github.com/nrwiersma/cluster/cluster"
)

// Application =============================

func newApplication(c *cmd.Context, agent cluster.Agent) (*cluster.Application, error) {
	app := cluster.NewApplication(cluster.Config{
		Agent:   agent,
		Logger:  c.Logger(),
		Statter: c.Statter(),
	})

	// Setup your application here

	return app, nil
}

// Agent ===================================

func newAgent(c *cmd.Context) (*clus.Agent, error) {
	cfg := clus.NewConfig()
	cfg.DataDir = c.String(flagDataDir)
	cfg.EncryptKey = c.String(flagEncryptKey)
	cfg.Bootstrap = c.Bool(flagBootstrap)
	cfg.BootstrapExpect = c.Int(flagBootstrapExpect)
	cfg.Logger = c.Logger()

	if id := c.String(flagID); id != "" {
		cfg.ID = id
	}

	if name := c.String(flagName); name != "" {
		cfg.Name = name
	}

	if advertiseAddr := c.String(flagRPCAdvertise); advertiseAddr != "" {
		addr, err := net.ResolveTCPAddr("tcp", advertiseAddr)
		if err != nil {
			return nil, fmt.Errorf("agent: invalid advertise address: %w", err)
		}
		cfg.RPCAdvertise = addr
	}

	if rpcAddr := c.String(flagRPCAddr); rpcAddr != "" {
		addr, err := net.ResolveTCPAddr("tcp", rpcAddr)
		if err != nil {
			return nil, fmt.Errorf("agent: invalid rpc address: %w", err)
		}
		cfg.RPCAddr = addr
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

	return clus.NewAgent(cfg)
}

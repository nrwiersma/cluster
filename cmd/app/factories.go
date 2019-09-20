package main

import (
	"github.com/hamba/cmd"
	"github.com/nrwiersma/cluster"
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

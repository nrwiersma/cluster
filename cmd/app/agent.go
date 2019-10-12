package main

import (
	"github.com/hamba/cmd"
	"gopkg.in/urfave/cli.v2"
)

func runAgent(c *cli.Context) error {
	ctx, err := cmd.NewContext(c)
	if err != nil {
		return err
	}

	agent, err := newAgent(ctx)
	if err != nil {
		return err
	}
	defer agent.Close()

	join := ctx.StringSlice(flagJoin)
	if len(join) > 0 {
		if err := agent.Join(join...); err != nil {
			return err
		}
	}

	app, err := newApplication(ctx, agent)
	if err != nil {
		return err
	}
	defer app.Close()

	<-cmd.WaitForSignals()

	if err := agent.Leave(); err != nil {
		return err
	}

	return nil
}

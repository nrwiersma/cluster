package cluster

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/hamba/pkg/log"
	"github.com/hamba/pkg/stats"
	"github.com/nrwiersma/cluster/cluster"
	"github.com/nrwiersma/cluster/cluster/rpc"
)

// Agent represents a cluster agent.
type Agent interface {
	AddLeaderRoutine(routine cluster.LeaderRoutine)
	Call(method string, req, resp interface{}) error
}

// Config configures an application.
type Config struct {
	Agent   Agent
	Logger  log.Logger
	Statter stats.Statter
}

// Application represents the application.
type Application struct {
	agent Agent

	shutdownCh chan struct{}

	logger  log.Logger
	statter stats.Statter
}

// NewApplication creates an instance of Application.
func NewApplication(cfg Config) *Application {
	app := &Application{
		agent:      cfg.Agent,
		shutdownCh: make(chan struct{}),
		logger:     cfg.Logger,
		statter:    cfg.Statter,
	}

	app.agent.AddLeaderRoutine(app.printNodes)

	return app
}

func (a *Application) printNodes(ctx context.Context) {
	a.logger.Info("I am the leader!")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Leadership lost, stopping!")
			return

		case <-ticker.C:
			req := rpc.NodesRequest{}
			var resp rpc.NodesResponse
			err := a.agent.Call("Cluster.GetNodes", &req, &resp)
			if err != nil {
				a.logger.Error("Error getting nodes", "error", err)
				continue
			}

			tw := tabwriter.NewWriter(os.Stdout, 10, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "")
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", "ID", "Name", "Role", "Address", "Health")
			for _, node := range resp.Nodes {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", node.ID, node.Name, node.Role, node.Address, node.Health)
			}
			fmt.Fprintln(tw, "")
			tw.Flush()
		}
	}
}

// Close closes the application.
func (a *Application) Close() error {
	close(a.shutdownCh)
	return nil
}

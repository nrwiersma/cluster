package cluster

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/hamba/pkg/log"
	"github.com/hamba/pkg/stats"
	"github.com/nrwiersma/cluster/cluster"
)

// Config configures an application.
type Config struct {
	Agent   *cluster.Agent
	DB      *DB
	Logger  log.Logger
	Statter stats.Statter
}

// Application represents the application.
type Application struct {
	agent *cluster.Agent
	db    *DB

	shutdownCh chan struct{}

	logger  log.Logger
	statter stats.Statter
}

// NewApplication creates an instance of Application.
func NewApplication(cfg Config) *Application {
	app := &Application{
		agent:      cfg.Agent,
		db:         cfg.DB,
		shutdownCh: make(chan struct{}),
		logger:     cfg.Logger,
		statter:    cfg.Statter,
	}

	go app.printNodes()

	return app
}

func (a *Application) printNodes() {
	for {
		nodes, watchCh, err := a.db.Nodes()
		if err != nil {
			a.logger.Error("app: error listing nodes", "error", err)
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', tabwriter.Debug)
		_, _ = fmt.Fprintf(w,
			"%s\t%s\n",
			"ID",
			"Health",
		)
		for _, node := range nodes {
			_, _ = fmt.Fprintf(w,
				"%s\t%s\n",
				node.ID,
				node.Health,
			)
		}
		_ = w.Flush()

		select {
		case <-watchCh:

		case <-a.db.AbandonCh():

		case <-a.shutdownCh:
			return
		}
	}
}

func (a *Application) Close() error {
	close(a.shutdownCh)
	return nil
}

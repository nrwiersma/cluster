package cluster

import (
	"fmt"
	"time"

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

	logger  log.Logger
	statter stats.Statter
}

// NewApplication creates an instance of Application.
func NewApplication(cfg Config) *Application {
	go func() {
		tick := time.NewTicker(10 * time.Second)
		for range tick.C {
			fmt.Println("NODES ============================")

			nodes, _ := cfg.DB.Nodes()
			for _, node := range nodes {
				fmt.Printf("%#v\n", node)
			}
			fmt.Println("==================================")
		}
	}()

	return &Application{
		agent:   cfg.Agent,
		db:      cfg.DB,
		logger:  cfg.Logger,
		statter: cfg.Statter,
	}
}

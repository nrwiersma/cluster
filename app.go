package cluster

import (
	"github.com/hamba/pkg/log"
	"github.com/hamba/pkg/stats"
)

// Agent represents a cluster agent.
type Agent interface {
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

	return app
}

func (a *Application) Close() error {
	close(a.shutdownCh)
	return nil
}

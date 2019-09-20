package cluster

import (
	"github.com/hamba/pkg/log"
	"github.com/hamba/pkg/stats"
)

// Application represents the application.
type Application struct {
	logger  log.Logger
	statter stats.Statter
}

// NewApplication creates an instance of Application.
func NewApplication(l log.Logger, s stats.Statter) *Application {
	return &Application{
		logger:  l,
		statter: s,
	}
}

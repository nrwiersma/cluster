package log

import (
	"io"
	stdlog "log"

	"github.com/hamba/pkg/log"
	"github.com/hashicorp/go-hclog"
)

// Level is the log level that will be used.
type Level int

// The log level constants.
const (
	Debug Level = iota
	Info
)

// Bridge is a log bridge to a standard logger.
type Bridge struct {
	log    log.Logger
	lvl    Level
	prefix string
}

// NewBridge returns a log bridge.
func NewBridge(l log.Logger, lvl Level, prefix string) *stdlog.Logger {
	adpt := &Bridge{
		log:    l,
		lvl:    lvl,
		prefix: prefix,
	}

	return stdlog.New(adpt, "", 0)
}

// Write writes a log line.
func (b *Bridge) Write(p []byte) (n int, err error) {
	line := b.prefix + string(p)

	switch b.lvl {
	case Debug:
		b.log.Debug(line)

	default:
		b.log.Info(line)
	}

	return len(p), nil
}

// Bridge is a log bridge to a hcl logger.
type HCLBridge struct {
	log    log.Logger
	prefix string
}

func NewHCLBridge(l log.Logger, prefix string) hclog.Logger {
	return &HCLBridge{
		log:    l,
		prefix: prefix,
	}
}

func (h *HCLBridge) Trace(msg string, args ...interface{}) {
	h.log.Debug(msg, args)
}

func (h *HCLBridge) Debug(msg string, args ...interface{}) {
	h.log.Debug(msg, args)
}

func (h *HCLBridge) Info(msg string, args ...interface{}) {
	h.log.Info(msg, args)
}

func (h *HCLBridge) Warn(msg string, args ...interface{}) {
	h.log.Info(msg, args)
}

func (h *HCLBridge) Error(msg string, args ...interface{}) {
	h.log.Error(msg, args)
}

func (h *HCLBridge) IsTrace() bool {
	return true
}

func (h *HCLBridge) IsDebug() bool {
	return true
}

func (h *HCLBridge) IsInfo() bool {
	return true
}

func (h *HCLBridge) IsWarn() bool {
	return true
}

func (h *HCLBridge) IsError() bool {
	return true
}

func (h *HCLBridge) With(args ...interface{}) hclog.Logger {
	// TODO: we need to do something here
	return h
}

func (h *HCLBridge) Named(name string) hclog.Logger {
	return h
}

func (h *HCLBridge) ResetNamed(name string) hclog.Logger {
	return h
}

func (h *HCLBridge) SetLevel(level hclog.Level) {}

func (h *HCLBridge) StandardLogger(opts *hclog.StandardLoggerOptions) *stdlog.Logger {
	return NewBridge(h.log, Debug, h.prefix)
}

func (h *HCLBridge) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return &Bridge{
		log:    h.log,
		lvl:    Debug,
		prefix: h.prefix,
	}
}

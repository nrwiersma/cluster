package cluster

import (
	"os"
	"time"

	"github.com/hamba/pkg/log"
	"github.com/hashicorp/raft"
	"github.com/hashicorp/serf/serf"
)

const (
	DefaultRPCAddr  = ":8300"
	DefaultSerfPort = 8301
)

// Config holds the configuration for a Agent.
type Config struct {
	// ID is a unique id for this agent.
	ID string

	// Name is the name the agent uses to advertise.
	Name string

	// DataDir is the directory to store our state in.
	DataDir string

	// SerfConfig is the configuration used from Serf.
	SerfConfig *serf.Config

	// EncryptKey is the encryption key used to secure
	// Serf communications. The entire cluster must use
	// the same encryption key.
	EncryptKey string

	// RaftConfig is the configuration used for Raft.
	RaftConfig *raft.Config

	// RPCAddr is the address used for RPC communication.
	RPCAddr string

	// Bootstrap is used to bring up the first cluster node.
	// This is required to create a single node cluster.
	Bootstrap bool

	// BootstrapExpect is the number of of nodes needed to
	// bootstrap the cluster, if bootstrapping is needed.
	BootstrapExpect int

	// NonVoter indicates that the node will not vote in the
	// the cluster. It will only receive state.
	NonVoter bool

	// LeaveDrainTime is the time to wait after leaving the cluster
	// to verify we actually left and drained connections.
	LeaveDrainTime time.Duration

	// ReconcileInterval controls how often we reconcile the strongly
	// consistent store with the Serf info.
	ReconcileInterval time.Duration

	// Logger is the logger to log to.
	Logger log.Logger

	// StartAsLeader starts the node as leader.
	// This should only be used for testing.
	StartAsLeader bool
}

// NewConfig creates/returns a default configuration.
func NewConfig() *Config {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	conf := &Config{
		Name:              hostname,
		SerfConfig:        serfDefaultConfig(),
		RaftConfig:        raft.DefaultConfig(),
		RPCAddr:           DefaultRPCAddr,
		LeaveDrainTime:    5 * time.Second,
		ReconcileInterval: 60 * time.Second,
	}

	conf.SerfConfig.ReconnectTimeout = 24 * time.Hour
	conf.SerfConfig.MemberlistConfig.BindPort = DefaultSerfPort

	conf.RaftConfig.SnapshotThreshold = 16384

	return conf
}

func serfDefaultConfig() *serf.Config {
	base := serf.DefaultConfig()
	base.QueueDepthWarning = 1000000
	return base
}

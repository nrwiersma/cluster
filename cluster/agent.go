package cluster

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hamba/pkg/log"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/hashicorp/serf/serf"
	"github.com/nrwiersma/cluster/cluster/fsm"
	"github.com/nrwiersma/cluster/cluster/rpc"
	"github.com/nrwiersma/cluster/cluster/state"
	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
)

const (
	serfSnapshot      = "serf/local.snapshot"
	raftState         = "raft/"
	raftLogCacheSize  = 512
	snapshotsRetained = 2
)

// Agent is a cluster server that manages Raft and Serf.
type Agent struct {
	config *Config
	log    log.Logger

	fsm *fsm.FSM

	raft          *raft.Raft
	raftStore     *raftboltdb.BoltStore
	raftTransport *raft.NetworkTransport
	raftNotifyCh  chan bool

	serf        *serf.Serf
	eventCh     chan serf.Event
	reconcileCh chan serf.Member

	shutdownMu sync.Mutex
	shutdownCh chan struct{}
	shutdown   bool
}

// NewAgent returns a new agent with the given configuration.
func NewAgent(cfg *Config) (*Agent, error) {
	if cfg.EncryptKey != "" {
		key, err := base64.StdEncoding.DecodeString(cfg.EncryptKey)
		if err != nil {
			return nil, errors.Wrap(err, "agent: failed to decode encryption key")
		}

		if err := memberlist.ValidateKey(key); err != nil {
			return nil, errors.Wrap(err, "agent: invalid encryption key")
		}

		cfg.SerfConfig.MemberlistConfig.SecretKey = key
	}

	if cfg.BootstrapExpect == 1 {
		cfg.Bootstrap = true
		cfg.BootstrapExpect = 0
	}

	logger := cfg.Logger
	if logger == nil {
		logger = log.Null
	}

	n := &Agent{
		config:       cfg,
		raftNotifyCh: make(chan bool, 1),
		eventCh:      make(chan serf.Event, 256),
		reconcileCh:  make(chan serf.Member, 32),
		shutdownCh:   make(chan struct{}),
		log:          logger,
	}

	if err := n.setupAgentID(); err != nil {
		return nil, errors.Wrap(err, "agent: error setting up agent id")
	}

	if err := n.setupRaft(); err != nil {
		n.Close()
		return nil, fmt.Errorf("agent: %v", err)
	}

	var err error
	n.serf, err = n.setupSerf(cfg.SerfConfig, n.eventCh, serfSnapshot)
	if err != nil {
		return nil, err
	}

	go n.eventHandler()

	go n.monitorLeadership()

	return n, nil
}

// DB returns the current state database.
//
// During a restore a new database will be created and
// the old database abandoned. If a reference to the
// database is kept, the AbandonCh should be watched
// and the new database fetched.
func (a *Agent) Store() *state.Store {
	return a.fsm.Store()
}

// IsLeader indicates if the agent is the leader of the cluster.
func (a *Agent) IsLeader() bool {
	return a.raft.State() == raft.Leader
}

// Members returns the members in the serf cluster.
func (a *Agent) Members() []serf.Member {
	return a.serf.Members()
}

// Join joins the cluster using the given Serf addresses.
func (a *Agent) Join(addrs ...string) error {
	if _, err := a.serf.Join(addrs, true); err != nil {
		return fmt.Errorf("agent: error joining cluster: %w", err)
	}
	return nil
}

// Leave leaves the cluster gracefully.
func (a *Agent) Leave() error {
	numPeers, err := a.numPeers()
	if err != nil {
		return errors.Wrap(err, "agent: check raft peers error")
	}

	isLeader := a.IsLeader()
	if isLeader && numPeers > 1 {
		future := a.raft.RemoveServer(raft.ServerID(a.config.ID), 0, 0)
		if err := future.Error(); err != nil {
			a.log.Error("agent: error removing ourselves as raft peer", "error", err)
		}
	}

	if a.serf != nil {
		if err := a.serf.Leave(); err != nil {
			return errors.Wrap(err, "agent: error leaving cluster")
		}
	}

	time.Sleep(a.config.LeaveDrainTime)

	// TODO: Other stuff here

	return nil
}

// Healthy determines if the agent is healthy.
func (a *Agent) Healthy() bool {
	if !a.IsLeader() && time.Since(a.raft.LastContact()) > time.Minute {
		return false
	}

	return true
}

// Close closes the agent.
// This is not a graceful shutdown. Call Leave to leave
// gracefully.
func (a *Agent) Close() error {
	a.shutdownMu.Lock()
	defer a.shutdownMu.Unlock()

	if a.shutdown {
		return nil
	}

	a.shutdown = true
	close(a.shutdownCh)

	if a.serf != nil {
		if err := a.serf.Shutdown(); err != nil {
			return fmt.Errorf("agent: error shutting down serf: %w", err)
		}
	}

	if a.raft != nil {
		_ = a.raftTransport.Close()
		future := a.raft.Shutdown()
		if err := future.Error(); err != nil {
			a.log.Error("agent: shutdown error", "error", err)
		}
		if a.raftStore != nil {
			_ = a.raftStore.Close()
		}
	}

	return nil
}

func (a *Agent) setupAgentID() error {
	if a.config.ID != "" {
		return nil
	}

	fileID := filepath.Join(a.config.DataDir, "node-id")
	if _, err := os.Stat(fileID); err == nil {
		rawID, err := ioutil.ReadFile(fileID)
		if err != nil {
			return err
		}

		id := strings.TrimSpace(string(rawID))
		if _, err := ksuid.Parse(id); err != nil {
			return err
		}

		a.config.ID = id
		return nil
	}

	id := ksuid.New().String()
	if err := ensurePath(fileID, false); err != nil {
		return err
	}
	if err := ioutil.WriteFile(fileID, []byte(id), 0600); err != nil {
		return err
	}
	a.config.ID = id

	return nil
}

func (a *Agent) raftApply(t rpc.MessageType, msg interface{}) (interface{}, error) {
	buf, err := rpc.Encode(t, msg)
	if err != nil {
		return nil, fmt.Errorf("agent: failed to encode request: %v", err)
	}

	future := a.raft.Apply(buf, 30*time.Second)
	if err := future.Error(); err != nil {
		return nil, err
	}
	return future.Response(), nil
}

func (a *Agent) numPeers() (int, error) {
	future := a.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return 0, err
	}

	raftConfig := future.Configuration()
	var numPeers int
	for _, server := range raftConfig.Servers {
		if server.Suffrage == raft.Voter {
			numPeers++
		}
	}

	return numPeers, nil
}

// ensurePath is used to make sure a path exists
func ensurePath(path string, dir bool) error {
	if !dir {
		path = filepath.Dir(path)
	}
	return os.MkdirAll(path, 0755)
}

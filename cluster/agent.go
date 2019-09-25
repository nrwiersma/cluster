package cluster

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hamba/pkg/log"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/hashicorp/serf/serf"
	"github.com/nrwiersma/cluster/cluster/fsm"
	"github.com/pkg/errors"
)

const (
	serfSnapshot      = "serf/local.snapshot"
	raftState         = "raft/"
	raftLogCacheSize  = 512
	snapshotsRetained = 2
)

type Agent struct {
	config *Config
	log    log.Logger

	raft          *raft.Raft
	raftStore     *raftboltdb.BoltStore
	raftTransport *raft.NetworkTransport
	fsm           *fsm.FSM
	raftNotifyCh  chan bool

	serf        *serf.Serf
	eventCh     chan serf.Event
	reconcileCh chan serf.Member

	shutdownMu sync.Mutex
	shutdownCh chan struct{}
	shutdown   bool
}

func New(cfg *Config) (*Agent, error) {
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

	if err := n.setupRaft(); err != nil {
		n.Close()
		return nil, fmt.Errorf("node: %v", err)
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

func (a *Agent) Members() []serf.Member {
	return a.serf.Members()
}

func (a *Agent) Join(addrs ...string) error {
	if _, err := a.serf.Join(addrs, true); err != nil {
		return fmt.Errorf("agent: error joining cluster: %w", err)
	}
	return nil
}

func (a *Agent) Leave() error {
	numPeers, err := a.numPeers()
	if err != nil {
		return errors.Wrap(err, "agent: check raft peers error")
	}

	isLeader := a.isLeader()
	if isLeader && numPeers > 1 {
		future := a.raft.RemoveServer(raft.ServerID(fmt.Sprintf("%d", a.config.ID)), 0, 0)
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

func (a *Agent) isLeader() bool {
	return a.raft.State() == raft.Leader
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

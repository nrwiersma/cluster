package cluster

import (
	"context"
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

type Server struct {
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

func New(cfg *Config) (*Server, error) {
	if cfg.EncryptKey != "" {
		key, err := base64.StdEncoding.DecodeString(cfg.EncryptKey)
		if err != nil {
			return nil, errors.Wrap(err, "server: failed to decode encryption key")
		}

		if err := memberlist.ValidateKey(key); err != nil {
			return nil, errors.Wrap(err, "server: invalid encryption key")
		}

		cfg.SerfConfig.MemberlistConfig.SecretKey = key
	}

	n := &Server{
		config:       cfg,
		raftNotifyCh: make(chan bool, 1),
		eventCh:      make(chan serf.Event, 256),
		reconcileCh:  make(chan serf.Member, 32),
		shutdownCh:   make(chan struct{}),
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

func (s *Server) Join(addrs ...string) error {
	if _, err := s.serf.Join(addrs, true); err != nil {
		return fmt.Errorf("cluster: error joining cluster: %w", err)
	}
	return nil
}

func (s *Server) Leave(ctx context.Context) error {
	numPeers, err := s.numPeers()
	if err != nil {
		return errors.Wrap(err, "cluster: check raft peers error")
	}

	isLeader := s.isLeader()
	if isLeader && numPeers > 1 {
		future := s.raft.RemoveServer(raft.ServerID(fmt.Sprintf("%d", s.config.NodeID)), 0, 0)
		if err := future.Error(); err != nil {
			s.log.Error("broker: remove ourself as raft peer error", "node-id", s.config.NodeID, "error", err)
		}
	}

	if s.serf != nil {
		if err := s.serf.Leave(); err != nil {
			return errors.Wrap(err, "cluster: error leaving cluster")
		}
	}

	time.Sleep(s.config.LeaveDrainTime)

	// TODO: Other stuff here

	return nil
}

func (s *Server) Close() error {
	s.shutdownMu.Lock()
	defer s.shutdownMu.Unlock()

	if s.shutdown {
		return nil
	}

	s.shutdown = true
	close(s.shutdownCh)

	if s.serf != nil {
		if err := s.serf.Shutdown(); err != nil {
			return fmt.Errorf("cluster: error shutting down serf: %w", err)
		}
	}

	return nil
}

func (s *Server) isLeader() bool {
	return s.raft.State() == raft.Leader
}

func (s *Server) numPeers() (int, error) {
	future := s.raft.GetConfiguration()
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

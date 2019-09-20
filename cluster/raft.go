package cluster

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/hashicorp/serf/serf"
	"github.com/nrwiersma/cluster/cluster/fsm"
	"github.com/nrwiersma/cluster/pkg/log"
)

const (
	barrierWriteTimeout = 2 * time.Minute
)

func (s *Server) setupRaft() error {
	s.config.RaftConfig.LocalID = raft.ServerID(fmt.Sprintf("%d", s.config.NodeID))
	s.config.RaftConfig.StartAsLeader = s.config.StartAsLeader // This is only for testing
	s.config.RaftConfig.NotifyCh = s.raftNotifyCh
	s.config.RaftConfig.Logger = log.NewHCLBridge(s.config.Logger, "raft: ")

	// Create the FSM
	s.fsm = fsm.New(s.config.NodeID)

	// Create the raft transport
	trans, err := raft.NewTCPTransportWithLogger(s.config.RPCAddr,
		nil,
		3,
		10*time.Second,
		log.NewBridge(s.config.Logger, log.Debug, "raft transport: "),
	)
	if err != nil {
		return err
	}
	s.raftTransport = trans

	path := filepath.Join(s.config.DataDir, raftState)
	if err := ensurePath(path, true); err != nil {
		return err
	}

	// Create the backend raft store for logs and stable storage.
	store, err := raftboltdb.NewBoltStore(filepath.Join(path, "raft.db"))
	if err != nil {
		return err
	}
	s.raftStore = store

	logStore, err := raft.NewLogCache(raftLogCacheSize, store)
	if err != nil {
		return err
	}

	snapshots, err := raft.NewFileSnapshotStore(path, snapshotsRetained, nil)
	if err != nil {
		return err
	}

	if s.config.Bootstrap {
		// We only need to bootstrap a single server at the start of a cluster
		hasState, err := raft.HasExistingState(logStore, store, snapshots)
		if err != nil {
			return err
		}
		if !hasState {
			configuration := raft.Configuration{
				Servers: []raft.Server{
					raft.Server{
						ID:      s.config.RaftConfig.LocalID,
						Address: trans.LocalAddr(),
					},
				},
			}
			if err := raft.BootstrapCluster(s.config.RaftConfig, logStore, store, snapshots, trans, configuration); err != nil {
				return err
			}
		}
	}

	s.raft, err = raft.NewRaft(s.config.RaftConfig, s.fsm, logStore, store, snapshots, trans)
	return err
}

// monitorLeadership monitors leadership change in raft, starting the leader loop
// when the server becomes leader.
func (s *Server) monitorLeadership() {
	var leaderLoopCh chan struct{}
	var leaderLoop sync.WaitGroup

	for {
		select {
		case leader := <-s.raftNotifyCh:
			if leader {
				if leaderLoopCh != nil {
					s.log.Error("leader: attempted to start the leader loop while running", "node-id", s.config.NodeID)
					continue
				}

				leaderLoopCh = make(chan struct{})
				leaderLoop.Add(1)
				go func(ch chan struct{}) {
					defer leaderLoop.Done()
					s.leaderLoop(ch)
				}(leaderLoopCh)

				s.log.Info("leader/%d: cluster leadership acquired", "node-id", s.config.NodeID)

				continue
			}

			if leaderLoopCh == nil {
				s.log.Error("leader: attempted to stop the leader loop while not running", "node-id", s.config.NodeID)
				continue
			}

			s.log.Debug("leader/%d: shutting down leader loop", "node-id", s.config.NodeID)

			close(leaderLoopCh)
			leaderLoop.Wait()
			leaderLoopCh = nil

			s.log.Info("leader: cluster leadership lost", "node-id", s.config.NodeID)

		case <-s.shutdownCh:
			return
		}
	}
}

// leaderLoop runs maintenaince tasks while the server is leader of the cluster.
func (s *Server) leaderLoop(stopCh chan struct{}) {
RECONCILE:
	interval := time.After(s.config.ReconcileInterval)
	barrier := s.raft.Barrier(barrierWriteTimeout)
	if err := barrier.Error(); err != nil {
		s.log.Error("leader: wait for barrier error", "node-id", s.config.NodeID, "error", err)
		goto WAIT
	}

	if err := s.reconcile(); err != nil {
		s.log.Error("leader: reconcile error", "node-id", s.config.NodeID, "error", err)
		goto WAIT
	}

WAIT:
	for {
		select {
		case <-stopCh:
			return
		case <-s.shutdownCh:
			return
		case <-interval:
			goto RECONCILE
		case member := <-s.reconcileCh:
			if err := s.reconcileMember(member); err != nil {
				s.log.Error("leader: reconcile member error", "node-id", s.config.NodeID, "error", err)
			}
		}
	}
}

func (s *Server) reconcile() error {
	panic("TODO")
}

func (s *Server) reconcileMember(m serf.Member) error {
	panic("TODO")
}

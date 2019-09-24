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

func (a *Agent) setupRaft() error {
	a.config.RaftConfig.LocalID = raft.ServerID(fmt.Sprintf("%d", a.config.ID))
	a.config.RaftConfig.StartAsLeader = a.config.StartAsLeader // This is only for testing
	a.config.RaftConfig.NotifyCh = a.raftNotifyCh
	a.config.RaftConfig.Logger = log.NewHCLBridge(a.config.Logger, "raft: ")

	// Create the FSM
	a.fsm = fsm.New(a.config.ID)

	// Create the raft transport
	trans, err := raft.NewTCPTransportWithLogger(a.config.RPCAddr,
		nil,
		3,
		10*time.Second,
		log.NewBridge(a.config.Logger, log.Debug, "raft transport: "),
	)
	if err != nil {
		return err
	}
	a.raftTransport = trans

	path := filepath.Join(a.config.DataDir, raftState)
	if err := ensurePath(path, true); err != nil {
		return err
	}

	// Create the backend raft store for logs and stable storage.
	store, err := raftboltdb.NewBoltStore(filepath.Join(path, "raft.db"))
	if err != nil {
		return err
	}
	a.raftStore = store

	logStore, err := raft.NewLogCache(raftLogCacheSize, store)
	if err != nil {
		return err
	}

	snapshots, err := raft.NewFileSnapshotStore(path, snapshotsRetained, nil)
	if err != nil {
		return err
	}

	if a.config.Bootstrap {
		// We only need to bootstrap a single server at the start of a cluster
		hasState, err := raft.HasExistingState(logStore, store, snapshots)
		if err != nil {
			return err
		}
		if !hasState {
			configuration := raft.Configuration{
				Servers: []raft.Server{
					raft.Server{
						ID:      a.config.RaftConfig.LocalID,
						Address: trans.LocalAddr(),
					},
				},
			}
			if err := raft.BootstrapCluster(a.config.RaftConfig, logStore, store, snapshots, trans, configuration); err != nil {
				return err
			}
		}
	}

	a.raft, err = raft.NewRaft(a.config.RaftConfig, a.fsm, logStore, store, snapshots, trans)
	return err
}

// monitorLeadership monitors leadership change in raft, starting the leader loop
// when the server becomes leader.
func (a *Agent) monitorLeadership() {
	var leaderLoopCh chan struct{}
	var leaderLoop sync.WaitGroup

	for {
		select {
		case leader := <-a.raftNotifyCh:
			if leader {
				if leaderLoopCh != nil {
					a.log.Error("leader: attempted to start the leader loop while running")
					continue
				}

				leaderLoopCh = make(chan struct{})
				leaderLoop.Add(1)
				go func(ch chan struct{}) {
					defer leaderLoop.Done()
					a.leaderLoop(ch)
				}(leaderLoopCh)

				a.log.Info("leader: cluster leadership acquired")

				continue
			}

			if leaderLoopCh == nil {
				a.log.Error("leader: attempted to stop the leader loop while not running")
				continue
			}

			a.log.Debug("leader: shutting down leader loop")

			close(leaderLoopCh)
			leaderLoop.Wait()
			leaderLoopCh = nil

			a.log.Info("leader: cluster leadership lost")

		case <-a.shutdownCh:
			return
		}
	}
}

// leaderLoop runs maintenaince tasks while the server is leader of the cluster.
func (a *Agent) leaderLoop(stopCh chan struct{}) {
RECONCILE:
	interval := time.After(a.config.ReconcileInterval)
	barrier := a.raft.Barrier(barrierWriteTimeout)
	if err := barrier.Error(); err != nil {
		a.log.Error("leader: wait for barrier error", "error", err)
		goto WAIT
	}

	if err := a.reconcile(); err != nil {
		a.log.Error("leader: reconcile error", "error", err)
		goto WAIT
	}

WAIT:
	for {
		select {
		case <-stopCh:
			return
		case <-a.shutdownCh:
			return
		case <-interval:
			goto RECONCILE
		case member := <-a.reconcileCh:
			if err := a.reconcileMember(member); err != nil {
				a.log.Error("leader: reconcile member error", "error", err)
			}
		}
	}
}

func (a *Agent) reconcile() error {
	panic("TODO")
}

func (a *Agent) reconcileMember(m serf.Member) error {
	panic("TODO")
}

package cluster

import (
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/hashicorp/serf/serf"
	"github.com/nrwiersma/cluster/cluster/fsm"
	"github.com/nrwiersma/cluster/cluster/metadata"
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
			a.reconcileMember(member)
		}
	}
}

func (a *Agent) reconcile() error {
	members := a.Members()
	knownMembers := make(map[string]struct{})
	for _, member := range members {
		a.reconcileMember(member)

		meta, ok := metadata.IsAgent(member)
		if !ok {
			continue
		}

		knownMembers[meta.ID.String()] = struct{}{}
	}

	return a.reconcileReaped(knownMembers)
}

func (a *Agent) reconcileReaped(known map[string]struct{}) error {
	future := a.raft.GetConfiguration()
	if future.Error() != nil {
		return future.Error()
	}

	raftConfig := future.Configuration()
	for _, server := range raftConfig.Servers {
		idStr := string(server.ID)
		if _, ok := known[idStr]; ok {
			continue
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		member := serf.Member{
			Tags: metadata.Agent{
				ID: metadata.NodeID(id),
			}.ToTags(),
		}
		if err := a.handleReapMember(member); err != nil {
			return err
		}
	}

	return nil
}

func (a *Agent) reconcileMember(m serf.Member) {
	var err error

	switch m.Status {
	case serf.StatusAlive:
		err = a.handleAliveMember(m)

	case serf.StatusFailed:
		err = a.handleFailedMember(m)

	case StatusReap:
		err = a.handleReapMember(m)

	case serf.StatusLeft:
		err = a.handleLeftMember(m)
	}

	if err != nil {
		a.log.Error("leader: reconcile member", "member", m.Name, "error", err)
	}
}

func (a *Agent) handleAliveMember(m serf.Member) error {
	agent, ok := metadata.IsAgent(m)
	if ok {
		if err := a.joinCluster(m, agent); err != nil {
			return err
		}
	}

	// TODO: tell someone about the node

	return nil
}

func (a *Agent) handleFailedMember(m serf.Member) error {
	//agent, ok := metadata.IsAgent(m)
	//if !ok {
	//	return nil
	//}

	// TODO: tell someone about the node

	return nil
}

func (a *Agent) handleLeftMember(m serf.Member) error {
	return a.handleDeregisterMember("left", m)
}

func (a *Agent) handleReapMember(member serf.Member) error {
	return a.handleDeregisterMember("reaped", member)
}

func (a *Agent) handleDeregisterMember(reason string, member serf.Member) error {
	agent, ok := metadata.IsAgent(member)
	if !ok {
		return nil
	}

	if agent.ID.Int32() == a.config.ID {
		a.log.Debug("leader: deregistering self should be done by follower")
		return nil
	}

	if err := a.removeServer(member, agent); err != nil {
		return err
	}

	// TODO: tell someone about the node

	return nil
}

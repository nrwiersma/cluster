package cluster

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/hashicorp/serf/serf"
	"github.com/nrwiersma/cluster/cluster/internal/fsm"
	"github.com/nrwiersma/cluster/cluster/internal/rpc"
	"github.com/nrwiersma/cluster/cluster/metadata"
	"github.com/nrwiersma/cluster/cluster/state"
	"github.com/nrwiersma/cluster/pkg/log"
)

const (
	barrierWriteTimeout = 2 * time.Minute
)

func (a *Agent) setupRaft() (err error) {
	// Protect against unclean exit
	defer func() {
		if a.raft == nil && a.raftStore != nil {
			_ = a.raftStore.Close()
		}
	}()

	a.config.RaftConfig.LocalID = raft.ServerID(a.config.ID)
	a.config.RaftConfig.StartAsLeader = a.config.StartAsLeader // This is only for testing
	a.config.RaftConfig.NotifyCh = a.raftNotifyCh
	a.config.RaftConfig.Logger = log.NewHCLBridge(a.config.Logger, "raft: ")

	// Create the FSM
	a.fsm, err = fsm.New()
	if err != nil {
		return err
	}

	// Create the raft transport
	trans := raft.NewNetworkTransportWithLogger(
		a.raftLayer,
		3,
		10*time.Second,
		log.NewBridge(a.config.Logger, log.Debug, "raft transport: "),
	)
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
					{
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
	var leaderStopCh chan struct{}
	var leaderLoop sync.WaitGroup

	for {
		select {
		case leader := <-a.raftNotifyCh:
			if leader {
				if leaderStopCh != nil {
					a.log.Error("leader: attempted to start the leader loop while running")
					continue
				}

				leaderStopCh = make(chan struct{})
				leaderLoop.Add(1)
				go func(ch chan struct{}) {
					defer leaderLoop.Done()
					a.leaderLoop(ch)
				}(leaderStopCh)

				a.raftRoutineManager.Start()

				a.log.Info("leader: cluster leadership acquired")

				continue
			}

			if leaderStopCh == nil {
				a.log.Error("leader: attempted to stop the leader loop while not running")
				continue
			}

			a.log.Debug("leader: shutting down leader loop")

			a.raftRoutineManager.Stop()
			close(leaderStopCh)
			leaderLoop.Wait()
			leaderStopCh = nil

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

		knownMembers[meta.ID] = struct{}{}
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
		id := string(server.ID)
		if _, ok := known[id]; ok {
			continue
		}

		member := serf.Member{
			Tags: metadata.Agent{ID: id}.ToTags(),
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

	case statusReap:
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
			a.log.Error("leader: error joining cluster", "member", agent.Name, "error", err)
			return err
		}
	}

	a.log.Info("leader: member joined, marking health alive", "member", m.Name)

	req := rpc.RegisterNodeRequest{
		Node: state.Node{
			ID:      m.Tags["id"],
			Name:    m.Name,
			Role:    m.Tags["role"],
			Address: m.Addr.String(),
			Health:  state.HealthPassing,
		},
	}
	if agent != nil {
		req.Node.Meta = m.Tags
	}

	_, err := a.raftApply(rpc.RegisterNodeRequestType, &req)
	return err
}

func (a *Agent) handleFailedMember(m serf.Member) error {
	agent, ok := metadata.IsAgent(m)
	if !ok {
		return nil
	}

	a.log.Info("leader: member failed, marking health critical", "member", m.Name)

	req := rpc.RegisterNodeRequest{
		Node: state.Node{
			ID:      m.Tags["id"],
			Name:    m.Name,
			Role:    m.Tags["role"],
			Address: m.Addr.String(),
			Health:  state.HealthCritical,
		},
	}
	if agent != nil {
		req.Node.Meta = m.Tags
	}
	_, err := a.raftApply(rpc.RegisterNodeRequestType, &req)
	return err
}

func (a *Agent) handleLeftMember(m serf.Member) error {
	return a.handleDeregisterMember("left", m)
}

func (a *Agent) handleReapMember(m serf.Member) error {
	return a.handleDeregisterMember("reaped", m)
}

func (a *Agent) handleDeregisterMember(reason string, m serf.Member) error {
	agent, ok := metadata.IsAgent(m)
	if !ok {
		return nil
	}

	if agent.ID == a.config.ID {
		a.log.Debug("leader: deregistering self should be done by follower")
		return nil
	}

	a.log.Info("leader: member left", "member", m.Name, "reason", reason)

	if err := a.removeServer(m, agent); err != nil {
		a.log.Error("leader: error joining cluster", "member", agent.Name, "error", err)
		return err
	}

	req := rpc.DeregisterNodeRequest{
		Node: state.Node{
			ID:   m.Tags["id"],
			Name: m.Name,
		},
	}
	_, err := a.raftApply(rpc.DeregisterNodeRequestType, &req)
	return err
}

func (a *Agent) joinCluster(m serf.Member, agent *metadata.Agent) error {
	if agent.Bootstrap {
		for _, member := range a.Members() {
			p, ok := metadata.IsAgent(member)
			if ok && member.Name != m.Name && p.Bootstrap {
				a.log.Error("leader: multiple nodes in bootstrap mode. there can only be one.")
				return nil
			}
		}
	}

	configFuture := a.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	// Processing ourselves could result in trying to remove ourselves to
	// fix up our address, which would make us step down. This is only
	// safe to attempt if there are multiple servers available.
	if m.Name == a.config.Name {
		if l := len(configFuture.Configuration().Servers); l < 3 {
			a.log.Debug("leader: skipping self join since cluster is too small", "servers", l)
			return nil
		}
	}

	for _, server := range configFuture.Configuration().Servers {
		if server.Address == raft.ServerAddress(agent.RPCAddr) || server.ID == raft.ServerID(agent.ID) {
			if server.Address == raft.ServerAddress(agent.RPCAddr) && server.ID == raft.ServerID(agent.ID) {
				// no-op if this is being called on an existing server
				return nil
			}

			future := a.raft.RemoveServer(server.ID, 0, 0)
			if server.Address == raft.ServerAddress(agent.RPCAddr) {
				if err := future.Error(); err != nil {
					return fmt.Errorf("leader: error removing server with duplicate address %q: %s", server.Address, err)
				}
				a.log.Info("leader: removed server with duplicated address", "address", server.Address)
			} else {
				if err := future.Error(); err != nil {
					return fmt.Errorf("leader: removing server with duplicate ID %q: %s", server.ID, err)
				}
				a.log.Info("leader: removed server with duplicate ID", "id", server.ID)
			}
		}
	}

	if agent.NonVoter {
		addFuture := a.raft.AddNonvoter(raft.ServerID(agent.ID), raft.ServerAddress(agent.RPCAddr), 0, 0)
		if err := addFuture.Error(); err != nil {
			return err
		}
		return nil
	}

	a.log.Debug("leader: join cluster", "voter", agent.ID)

	addFuture := a.raft.AddVoter(raft.ServerID(agent.ID), raft.ServerAddress(agent.RPCAddr), 0, 0)
	if err := addFuture.Error(); err != nil {
		return err
	}

	return nil
}

func (a *Agent) removeServer(m serf.Member, agent *metadata.Agent) error {
	configFuture := a.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}

	for _, server := range configFuture.Configuration().Servers {
		if server.ID != raft.ServerID(agent.ID) {
			continue
		}

		a.log.Info("leader: removing server by id", "id", server.ID)

		future := a.raft.RemoveServer(raft.ServerID(agent.ID), 0, 0)
		if err := future.Error(); err != nil {
			return err
		}
	}
	return nil
}

// LeaderRoutine is a routine to be run when acquiring leadership
type LeaderRoutine func(ctx context.Context)

// LeaderRoutineManager manages routines to run when leadership is
// acquired.
type LeaderRoutineManager struct {
	mu       sync.Mutex
	routines []LeaderRoutine
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
}

// Register registers a routine to run.
func (m *LeaderRoutineManager) Register(routine LeaderRoutine) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.routines = append(m.routines, routine)

	if m.running {
		go routine(m.ctx)
	}
}

// Start starts the registered routines.
func (m *LeaderRoutineManager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return
	}
	m.running = true

	m.ctx, m.cancel = context.WithCancel(context.Background())
	for _, routine := range m.routines {
		go routine(m.ctx)
	}
}

// Stop signals all the routines to stop.
func (m *LeaderRoutineManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return
	}
	m.running = false

	m.cancel()
	m.cancel = nil
	m.ctx = nil
}

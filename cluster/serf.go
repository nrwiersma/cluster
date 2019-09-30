package cluster

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/raft"
	"github.com/hashicorp/serf/serf"
	"github.com/nrwiersma/cluster/cluster/metadata"
	"github.com/nrwiersma/cluster/pkg/log"
)

const statusReap = serf.MemberStatus(-1)

func (a *Agent) setupSerf(config *serf.Config, ch chan serf.Event, path string) (*serf.Serf, error) {
	config.Init()
	config.NodeName = a.config.Name
	config.Tags = metadata.Agent{
		ID:        a.config.ID,
		Name:      a.config.Name,
		Bootstrap: a.config.Bootstrap,
		Expect:    a.config.BootstrapExpect,
		NonVoter:  a.config.NonVoter,
		SerfAddr:  fmt.Sprintf("%s:%d", a.config.SerfConfig.MemberlistConfig.BindAddr, a.config.SerfConfig.MemberlistConfig.BindPort),
		RPCAddr:   a.config.RPCAdvertise.String(),
	}.ToTags()
	config.Logger = log.NewBridge(a.config.Logger, log.Debug, "serf: ")
	config.MemberlistConfig.Logger = log.NewBridge(a.config.Logger, log.Debug, "memberlist: ")
	config.EventCh = ch
	config.EnableNameConflictResolution = false
	config.SnapshotPath = filepath.Join(a.config.DataDir, path)

	if err := ensurePath(config.SnapshotPath, false); err != nil {
		return nil, err
	}

	return serf.Create(config)
}

func (a *Agent) eventHandler() {
	for {
		select {
		case e := <-a.eventCh:
			switch e.EventType() {
			case serf.EventMemberJoin:
				a.nodeJoin(e.(serf.MemberEvent))
				a.localMemberEvent(e.(serf.MemberEvent))
			case serf.EventMemberReap:
				a.localMemberEvent(e.(serf.MemberEvent))
			case serf.EventMemberLeave, serf.EventMemberFailed:
				a.nodeFailed(e.(serf.MemberEvent))
				a.localMemberEvent(e.(serf.MemberEvent))
			}

		case <-a.shutdownCh:
			return
		}
	}
}

func (a *Agent) nodeJoin(e serf.MemberEvent) {
	for _, m := range e.Members {
		agent, ok := metadata.IsAgent(m)
		if !ok {
			continue
		}

		a.log.Info("agent: adding agent", "id", agent.ID)

		//a.brokerLookup.AddBroker(agent)
		if a.config.BootstrapExpect != 0 {
			a.maybeBootstrap()
		}
	}
}

func (a *Agent) nodeFailed(e serf.MemberEvent) {
	for _, m := range e.Members {
		agent, ok := metadata.IsAgent(m)
		if !ok {
			continue
		}

		a.log.Info("agent: removing agent", "id", agent.ID)

		//a.brokerLookup.RemoveBroker(agent)
	}
}

func (a *Agent) localMemberEvent(e serf.MemberEvent) {
	if !a.IsLeader() {
		return
	}

	isReap := e.EventType() == serf.EventMemberReap

	for _, m := range e.Members {
		if isReap {
			m.Status = statusReap
		}

		select {
		case a.reconcileCh <- m:
		default:
		}
	}
}

func (a *Agent) maybeBootstrap() {
	index, err := a.raftStore.LastIndex()
	if err != nil {
		a.log.Error("agent: read last raft index error", "error", err)
		return
	}
	if index != 0 {
		a.log.Info("agent: raft data found, disabling bootstrap mode", "index", index, "path", filepath.Join(a.config.DataDir, raftState))
		a.config.BootstrapExpect = 0
		return
	}

	members := a.Members()
	agents := make([]metadata.Agent, 0, len(members))
	for _, member := range members {
		agent, ok := metadata.IsAgent(member)
		if !ok {
			continue
		}

		if agent.Expect != 0 && agent.Expect != a.config.BootstrapExpect {
			a.log.Error("agent: members expects conflicting node count", "member", member.Name)
			return
		}

		if agent.Bootstrap {
			a.log.Error("agent: member has bootstrap mode. expect disabled", "member", member.Name)
			return
		}

		agents = append(agents, *agent)
	}

	if len(agents) < a.config.BootstrapExpect {
		a.log.Debug(fmt.Sprintf("agent: maybe bootstrap: need more brokers: got: %d: expect: %d", len(agents), a.config.BootstrapExpect))
		return
	}

	var configuration raft.Configuration
	addrs := make([]string, 0, len(agents))
	for _, agent := range agents {
		addr := agent.RPCAddr
		addrs = append(addrs, addr)
		peer := raft.Server{
			ID:      raft.ServerID(agent.ID),
			Address: raft.ServerAddress(addr),
		}
		configuration.Servers = append(configuration.Servers, peer)
	}

	a.log.Info("agent: found expected number of peers, attempting bootstrap", "addrs", addrs)
	future := a.raft.BootstrapCluster(configuration)
	if err := future.Error(); err != nil {
		a.log.Error("agent: bootstrap cluster error", "error", err)
	}
	a.config.BootstrapExpect = 0
}

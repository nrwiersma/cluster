package cluster

import (
	"sync"

	"github.com/hashicorp/raft"
	"github.com/nrwiersma/cluster/cluster/metadata"
)

type agentLookup struct {
	mu             sync.RWMutex
	addressToAgent map[raft.ServerAddress]*metadata.Agent
	idToAgent      map[raft.ServerID]*metadata.Agent
}

func newAgentLookup() *agentLookup {
	return &agentLookup{
		addressToAgent: make(map[raft.ServerAddress]*metadata.Agent),
		idToAgent:      make(map[raft.ServerID]*metadata.Agent),
	}
}

// Add adds an agent to the lookup.
func (l *agentLookup) Add(agent *metadata.Agent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.addressToAgent[raft.ServerAddress(agent.RPCAddr)] = agent
	l.idToAgent[raft.ServerID(agent.ID)] = agent
}

// Remove removes an agent from the lookup.
func (l *agentLookup) Remove(agent *metadata.Agent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.addressToAgent, raft.ServerAddress(agent.RPCAddr))
	delete(l.idToAgent, raft.ServerID(agent.ID))
}

// AgentByAddr looks up the agent by address.
func (l *agentLookup) AgentByAddr(addr raft.ServerAddress) *metadata.Agent {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.addressToAgent[addr]
}

// AgentByID looks up the agent by id.
func (l *agentLookup) AgentByID(id raft.ServerID) *metadata.Agent {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.idToAgent[id]
}

// Agents returns all agents.
func (l *agentLookup) Agents() []*metadata.Agent {
	l.mu.RLock()
	defer l.mu.RUnlock()

	agents := make([]*metadata.Agent, 0, len(l.idToAgent))
	for _, agent := range l.idToAgent {
		agents = append(agents, agent)
	}
	return agents
}

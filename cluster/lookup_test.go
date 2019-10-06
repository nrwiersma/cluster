package cluster

import (
	"testing"

	"github.com/nrwiersma/cluster/cluster/metadata"
	"github.com/stretchr/testify/assert"
)

func TestAgentLookup_Add(t *testing.T) {
	agent := &metadata.Agent{
		ID:      "test",
		RPCAddr: "127.0.0.1:8080",
	}

	l := newAgentLookup()

	l.Add(agent)

	assert.Len(t, l.idToAgent, 1)
	assert.Equal(t, agent, l.idToAgent["test"])
	assert.Len(t, l.addressToAgent, 1)
	assert.Equal(t, agent, l.addressToAgent["127.0.0.1:8080"])
}

func TestAgentLookup_Remove(t *testing.T) {
	agent := &metadata.Agent{
		ID:      "test",
		RPCAddr: "127.0.0.1:8080",
	}

	l := newAgentLookup()
	l.Add(agent)

	l.Remove(agent)

	assert.Len(t, l.idToAgent, 0)
	assert.Len(t, l.addressToAgent, 0)
}

func TestAgentLookup_AgentByAddr(t *testing.T) {
	agent := &metadata.Agent{
		ID:      "test",
		RPCAddr: "127.0.0.1:8080",
	}

	l := newAgentLookup()
	l.Add(agent)

	got := l.AgentByAddr("127.0.0.1:8080")

	assert.Equal(t, agent, got)
}

func TestAgentLookup_AgentByID(t *testing.T) {
	agent := &metadata.Agent{
		ID:      "test",
		RPCAddr: "127.0.0.1:8080",
	}

	l := newAgentLookup()
	l.Add(agent)

	got := l.AgentByID("test")

	assert.Equal(t, agent, got)
}

func TestAgentLookup_Agents(t *testing.T) {
	agent1 := &metadata.Agent{
		ID:      "test1",
		RPCAddr: "127.0.0.1:8080",
	}
	agent2 := &metadata.Agent{
		ID:      "test2",
		RPCAddr: "127.0.0.1:8081",
	}

	l := newAgentLookup()
	l.Add(agent1)
	l.Add(agent2)

	got := l.Agents()

	assert.Contains(t, got, agent1)
	assert.Contains(t, got, agent2)
}

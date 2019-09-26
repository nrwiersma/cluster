package metadata

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/serf/serf"
)

const (
	clusterName = "test"
)

// Agent is an cluster agent with its configuration.
type Agent struct {
	ID        string
	Name      string
	Bootstrap bool
	Expect    int
	NonVoter  bool
	Status    serf.MemberStatus
	SerfAddr  string
	RPCAddr   string
}

// ToTags converts the agent information into serf member tags.
func (a Agent) ToTags() map[string]string {
	tags := map[string]string{
		"cluster":   clusterName,
		"id":        a.ID,
		"serf_addr": a.SerfAddr,
		"rpc_addr":  a.RPCAddr,
	}

	if a.Bootstrap {
		tags["bootstrap"] = "1"
	}
	if a.Expect != 0 {
		tags["expect"] = fmt.Sprintf("%d", a.Expect)
	}
	if a.NonVoter {
		tags["non_voter"] = "1"
	}

	return tags
}

// IsAgent checks if the given serf member is an agent.
func IsAgent(m serf.Member) (*Agent, bool) {
	if m.Tags["cluster"] != clusterName {
		return nil, false
	}

	expect := 0
	expectStr, ok := m.Tags["expect"]
	var err error
	if ok {
		expect, err = strconv.Atoi(expectStr)
		if err != nil {
			return nil, false
		}
	}

	_, bootstrap := m.Tags["bootstrap"]
	_, nonVoter := m.Tags["non_voter"]

	return &Agent{
		ID:        m.Tags["id"],
		Name:      m.Tags["name"],
		Bootstrap: bootstrap,
		Expect:    expect,
		NonVoter:  nonVoter,
		Status:    m.Status,
		SerfAddr:  m.Tags["serf_lan_addr"],
		RPCAddr:   m.Tags["rpc_addr"],
	}, true
}

// AgentTags is used to create tags from a raft server.
func AgentTags(id string) map[string]string {
	return map[string]string{
		"cluster": clusterName,
		"id":      id,
	}
}

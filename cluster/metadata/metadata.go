package metadata

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/serf/serf"
)

type NodeID int32

func (n NodeID) Int32() int32 {
	return int32(n)
}

func (n NodeID) String() string {
	return fmt.Sprintf("%d", n)
}

type Agent struct {
	ID        NodeID
	Name      string
	Bootstrap bool
	Expect    int
	NonVoter  bool
	Status    serf.MemberStatus
	RaftAddr  string
	SerfAddr  string
}

// ToTags converts the agent information into serf member tags.
func (a *Agent) ToTags() map[string]string {
	tags := map[string]string{
		"cluster":   "test",
		"id":        fmt.Sprintf("%d", a.ID),
		"raft_addr": a.RaftAddr,
		"serf_addr": a.SerfAddr,
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
	if m.Tags["cluster"] != "test" {
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

	idStr := m.Tags["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, false
	}

	return &Agent{
		ID:        NodeID(id),
		Name:      m.Tags["name"],
		Bootstrap: bootstrap,
		Expect:    expect,
		NonVoter:  nonVoter,
		Status:    m.Status,
		RaftAddr:  m.Tags["raft_addr"],
		SerfAddr:  m.Tags["serf_lan_addr"],
	}, true
}

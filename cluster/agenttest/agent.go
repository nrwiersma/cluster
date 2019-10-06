package agenttest

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hamba/testutils/retry"
	"github.com/nrwiersma/cluster/cluster"
	"github.com/travisjeffery/go-dynaport"
)

var (
	nodeNumber int32
)

// NewAgent creates a test agent.
func NewAgent(t *testing.T, cfgFn func(cfg *cluster.Config)) (*cluster.Agent, *cluster.Config, string) {
	ports := dynaport.Get(2)
	id := atomic.AddInt32(&nodeNumber, 1)

	tmpDir, err := ioutil.TempDir("", fmt.Sprintf("cluster-test-server-%d", id))
	if err != nil {
		panic(err)
	}

	config := cluster.NewConfig()
	config.Name = fmt.Sprintf("%s-node-%d", t.Name(), id)
	config.DataDir = tmpDir
	config.RPCAddr = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: ports[0]}
	config.SerfConfig.MemberlistConfig.BindAddr = "127.0.0.1"
	config.SerfConfig.MemberlistConfig.BindPort = ports[1]
	config.LeaveDrainTime = 1 * time.Millisecond
	config.ReconcileInterval = 300 * time.Millisecond

	// Tighten the Serf timing
	config.SerfConfig.MemberlistConfig.BindAddr = "127.0.0.1"
	config.SerfConfig.MemberlistConfig.SuspicionMult = 2
	config.SerfConfig.MemberlistConfig.RetransmitMult = 2
	config.SerfConfig.MemberlistConfig.ProbeTimeout = 50 * time.Millisecond
	config.SerfConfig.MemberlistConfig.ProbeInterval = 100 * time.Millisecond
	config.SerfConfig.MemberlistConfig.GossipInterval = 100 * time.Millisecond

	// Tighten the Raft timing
	config.RaftConfig.LeaderLeaseTimeout = 100 * time.Millisecond
	config.RaftConfig.HeartbeatTimeout = 200 * time.Millisecond
	config.RaftConfig.ElectionTimeout = 200 * time.Millisecond

	if cfgFn != nil {
		cfgFn(config)
	}

	agent, err := cluster.NewAgent(config)
	if err != nil {
		t.Fatalf("err != nil: %s", err)
	}

	return agent, config, tmpDir
}

// CloseAndRemove closes an agent and removes its temp directory.
func CloseAndRemove(t *testing.T, agent *cluster.Agent, tmpDir string) {
	defer os.RemoveAll(tmpDir)

	if err := agent.Close(); err != nil {
		t.Error(fmt.Errorf("error closing agent: %w", err))
	}
}

// Join joins test agents.
func Join(t *testing.T, cfg *cluster.Config, agents ...*cluster.Agent) {
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.SerfConfig.MemberlistConfig.BindPort)
	for _, a := range agents {
		if err := a.Join(addr); err != nil {
			t.Fatalf("join err: %v", err)
		}
	}
}

// WaitForLeader waits for one of the agents to be leader, failing the test if no one is the leader.
// Returns the leader (if there is one) and non-leaders.
func WaitForLeader(t *testing.T, agents ...*cluster.Agent) (*cluster.Agent, []*cluster.Agent) {
	tmp := struct {
		leader    *cluster.Agent
		followers map[*cluster.Agent]bool
	}{nil, make(map[*cluster.Agent]bool)}

	retry.Run(t, func(t *retry.SubT) {
		for _, a := range agents {
			if a.IsLeader() {
				tmp.leader = a
				tmp.followers[a] = false
			} else {
				tmp.followers[a] = true
			}
		}

		if tmp.leader == nil {
			t.Fatal("no leader")
		}
	})

	followers := make([]*cluster.Agent, 0, len(tmp.followers))
	for f, ok := range tmp.followers {
		if !ok {
			continue
		}

		followers = append(followers, f)
	}
	return tmp.leader, followers
}

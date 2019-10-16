package cluster_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hamba/testutils/retry"
	"github.com/nrwiersma/cluster/cluster"
	"github.com/nrwiersma/cluster/cluster/agenttest"
	"github.com/nrwiersma/cluster/cluster/rpc"
	"github.com/nrwiersma/cluster/cluster/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent_Join(t *testing.T) {
	a1, cfg1, dir1 := agenttest.NewAgent(t, nil)
	defer agenttest.CloseAndRemove(t, a1, dir1)

	a2, _, dir2 := agenttest.NewAgent(t, nil)
	defer agenttest.CloseAndRemove(t, a2, dir2)

	agenttest.Join(t, cfg1, a2)

	require.Equal(t, 2, len(a1.Members()))
	require.Equal(t, 2, len(a2.Members()))
}

func TestAgent_CanRegisterMembers(t *testing.T) {
	a1, cfg1, dir1 := agenttest.NewAgent(t, func(cfg *cluster.Config) {
		cfg.Bootstrap = true
		cfg.BootstrapExpect = 3
	})
	defer agenttest.CloseAndRemove(t, a1, dir1)

	a2, cfg2, dir2 := agenttest.NewAgent(t, func(cfg *cluster.Config) {
		cfg.Bootstrap = false
		cfg.BootstrapExpect = 3
	})
	defer agenttest.CloseAndRemove(t, a2, dir2)

	agenttest.Join(t, cfg2, a1)
	agenttest.WaitForLeader(t, a1, a2)

	retry.Run(t, func(t *retry.SubT) {
		req := rpc.NodesRequest{
			Filter: fmt.Sprintf("ID == `%s`", cfg1.ID),
		}
		var resp rpc.NodesResponse
		err := a1.Call("Cluster.GetNodes", &req, &resp)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(resp.Nodes) != 1 {
			t.Fatal("node not registered")
		}
	})
	retry.Run(t, func(t *retry.SubT) {
		req := rpc.NodesRequest{
			Filter: fmt.Sprintf("ID == `%s`", cfg2.ID),
		}
		var resp rpc.NodesResponse
		err := a1.Call("Cluster.GetNodes", &req, &resp)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(resp.Nodes) != 1 {
			t.Fatal("node not registered")
		}
	})
}

func TestAgent_HandlesFailedMember(t *testing.T) {
	a1, _, dir1 := agenttest.NewAgent(t, func(cfg *cluster.Config) {
		cfg.Bootstrap = true
		cfg.BootstrapExpect = 1
		cfg.StartAsLeader = true
	})
	defer agenttest.CloseAndRemove(t, a1, dir1)

	a2, cfg2, dir2 := agenttest.NewAgent(t, func(cfg *cluster.Config) {
		cfg.Bootstrap = false
		cfg.NonVoter = true
	})
	defer os.RemoveAll(dir2)

	agenttest.Join(t, cfg2, a1)

	_ = a2.Close()

	retry.Run(t, func(t *retry.SubT) {
		req := rpc.NodesRequest{
			Filter: fmt.Sprintf("ID == `%s`", cfg2.ID),
		}
		var resp rpc.NodesResponse
		err := a1.Call("Cluster.GetNodes", &req, &resp)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(resp.Nodes) != 1 {
			t.Fatal("node not registered")
		}
		if resp.Nodes[0].Health == state.HealthCritical {
			t.Fatal("node not critical")
		}
	})
}

func TestAgent_HandlesLeftMember(t *testing.T) {
	a1, _, dir1 := agenttest.NewAgent(t, func(cfg *cluster.Config) {
		cfg.Bootstrap = true
		cfg.BootstrapExpect = 1
		cfg.StartAsLeader = true
	})
	defer agenttest.CloseAndRemove(t, a1, dir1)

	a2, cfg2, dir2 := agenttest.NewAgent(t, func(cfg *cluster.Config) {
		cfg.Bootstrap = false
		cfg.NonVoter = true
	})
	defer agenttest.CloseAndRemove(t, a2, dir2)

	agenttest.Join(t, cfg2, a1)
	agenttest.WaitForLeader(t, a1, a2)

	req := rpc.NodesRequest{
		Filter: fmt.Sprintf("ID == `%s`", cfg2.ID),
	}

	// Should be registered
	retry.Run(t, func(t *retry.SubT) {
		var resp rpc.NodesResponse
		err := a1.Call("Cluster.GetNodes", &req, &resp)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(resp.Nodes) != 1 {
			t.Fatal("node isn't registered")
		}
	})

	err := a2.Leave()
	require.NoError(t, err)

	// Should be deregistered
	retry.Run(t, func(t *retry.SubT) {
		var resp rpc.NodesResponse
		err := a1.Call("Cluster.GetNodes", &req, &resp)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(resp.Nodes) != 0 {
			t.Fatal("node still registered")
		}
	})
}

func TestAgent_HandlesLeftLeader(t *testing.T) {
	a1, cfg1, dir1 := agenttest.NewAgent(t, func(cfg *cluster.Config) {
		cfg.Bootstrap = false
		cfg.BootstrapExpect = 3
	})
	defer agenttest.CloseAndRemove(t, a1, dir1)

	a2, cfg2, dir2 := agenttest.NewAgent(t, func(cfg *cluster.Config) {
		cfg.Bootstrap = false
		cfg.BootstrapExpect = 3
	})
	defer agenttest.CloseAndRemove(t, a2, dir2)

	a3, cfg3, dir3 := agenttest.NewAgent(t, func(cfg *cluster.Config) {
		cfg.Bootstrap = false
		cfg.BootstrapExpect = 3
	})
	defer agenttest.CloseAndRemove(t, a3, dir3)

	agentCfgs := map[*cluster.Agent]*cluster.Config{a1: cfg1, a2: cfg2, a3: cfg3}

	agenttest.Join(t, cfg2, a1)
	agenttest.Join(t, cfg3, a1)

	leader, followers := agenttest.WaitForLeader(t, a1, a2, a3)
	require.NotNil(t, leader)

	var leaderCfg *cluster.Config
	for a, cfg := range agentCfgs {
		if a == leader {
			leaderCfg = cfg
		}
	}

	err := leader.Leave()
	require.NoError(t, err)
	_ = leader.Close()

	leader2, _ := agenttest.WaitForLeader(t, followers...)
	require.NotNil(t, leader2)

	retry.Run(t, func(t *retry.SubT) {
		req := rpc.NodesRequest{
			Filter: fmt.Sprintf("ID == `%s`", leaderCfg.ID),
		}
		var resp rpc.NodesResponse
		err := leader2.Call("Cluster.GetNodes", &req, &resp)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(resp.Nodes) != 0 {
			t.Fatal("leader should be deregistered")
		}
	})
}

func TestAgent_Close(t *testing.T) {
	a, _, dir := agenttest.NewAgent(t, func(cfg *cluster.Config) {
		cfg.Bootstrap = true
	})
	defer os.RemoveAll(dir)

	err := a.Close()

	assert.NoError(t, err)
}

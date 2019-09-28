package cluster_test

import (
	"os"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil/retry"
	"github.com/nrwiersma/cluster/cluster"
	"github.com/nrwiersma/cluster/cluster/agenttest"
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

	store := a1.Store()
	retry.Run(t, func(r *retry.R) {
		node, err := store.Node(cfg1.ID)
		if err != nil {
			r.Fatalf("err: %v", err)
		}
		if node == nil {
			r.Fatal("node not registered")
		}
	})
	retry.Run(t, func(r *retry.R) {
		node, err := store.Node(cfg2.ID)
		if err != nil {
			r.Fatalf("err: %v", err)
		}
		if node == nil {
			r.Fatal("node not registered")
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

	store := a1.Store()

	retry.Run(t, func(r *retry.R) {
		node, err := store.Node(cfg2.ID)
		if err != nil {
			r.Fatalf("err: %v", err)
		}
		if node == nil {
			r.Fatal("node not registered")
		}
		if node.Health == state.HealthCritical {
			r.Fatal("node not critical")
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

	store := a1.Store()

	// Should be registered
	retry.Run(t, func(r *retry.R) {
		node, err := store.Node(cfg2.ID)
		if err != nil {
			r.Fatalf("err: %v", err)
		}
		if node == nil {
			r.Fatal("node isn't registered")
		}
	})

	err := a2.Leave()
	require.NoError(t, err)

	// Should be deregistered
	retry.Run(t, func(r *retry.R) {
		node, err := store.Node(cfg2.ID)
		if err != nil {
			r.Fatalf("err: %v", err)
		}
		if node != nil {
			r.Fatal("node still registered")
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

	store := leader2.Store()
	retry.Run(t, func(r *retry.R) {
		node, err := store.Node(leaderCfg.ID)
		if err != nil {
			r.Fatalf("err: %v", err)
		}
		if node != nil {
			r.Fatal("leader should be deregistered")
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

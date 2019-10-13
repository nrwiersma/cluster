package metadata_test

import (
	"testing"

	"github.com/hashicorp/serf/serf"
	"github.com/nrwiersma/cluster/cluster/metadata"
	"github.com/stretchr/testify/assert"
)

func TestAgent_ToTags(t *testing.T) {
	tests := []struct {
		name  string
		agent metadata.Agent
		want  map[string]string
	}{
		{
			name: "Full Agent",
			agent: metadata.Agent{
				ID:        "id",
				Name:      "name",
				Bootstrap: true,
				Expect:    3,
				NonVoter:  true,
				SerfAddr:  "serf-addr",
				RPCAddr:   "rpc-addr",
			},
			want: map[string]string{
				"cluster":   "test",
				"id":        "id",
				"name":      "name",
				"role":      "agent",
				"bootstrap": "1",
				"expect":    "3",
				"non_voter": "1",
				"rpc_addr":  "rpc-addr",
				"serf_addr": "serf-addr",
			},
		},
		{
			name: "False Bootstrap",
			agent: metadata.Agent{
				ID:        "id",
				Name:      "name",
				Bootstrap: false,
				Expect:    3,
				NonVoter:  true,
				SerfAddr:  "serf-addr",
				RPCAddr:   "rpc-addr",
			},
			want: map[string]string{
				"cluster":   "test",
				"id":        "id",
				"name":      "name",
				"role":      "agent",
				"expect":    "3",
				"non_voter": "1",
				"rpc_addr":  "rpc-addr",
				"serf_addr": "serf-addr",
			},
		},
		{
			name: "Full Agent",
			agent: metadata.Agent{
				ID:        "id",
				Name:      "name",
				Bootstrap: true,
				Expect:    3,
				NonVoter:  false,
				SerfAddr:  "serf-addr",
				RPCAddr:   "rpc-addr",
			},
			want: map[string]string{
				"cluster":   "test",
				"id":        "id",
				"name":      "name",
				"role":      "agent",
				"bootstrap": "1",
				"expect":    "3",
				"rpc_addr":  "rpc-addr",
				"serf_addr": "serf-addr",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.agent.ToTags()

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsAgent(t *testing.T) {
	tests := []struct {
		name   string
		member serf.Member
		want   *metadata.Agent
		wantOk bool
	}{
		{
			name: "Full Agent",
			member: serf.Member{
				Tags: map[string]string{
					"cluster":   "test",
					"id":        "id",
					"name":      "name",
					"role":      "agent",
					"bootstrap": "1",
					"expect":    "3",
					"non_voter": "1",
					"rpc_addr":  "rpc-addr",
					"serf_addr": "serf-addr",
				},
				Status: serf.StatusAlive,
			},
			want: &metadata.Agent{
				ID:        "id",
				Name:      "name",
				Bootstrap: true,
				Expect:    3,
				NonVoter:  true,
				Status:    serf.StatusAlive,
				RPCAddr:   "rpc-addr",
				SerfAddr:  "serf-addr",
			},
			wantOk: true,
		},
		{
			name: "Is Agent",
			member: serf.Member{
				Tags: map[string]string{
					"cluster":   "test",
					"id":        "id",
					"name":      "name",
					"role":      "agent",
					"expect":    "3",
					"rpc_addr":  "rpc-addr",
					"serf_addr": "serf-addr",
				},
				Status: serf.StatusAlive,
			},
			want: &metadata.Agent{
				ID:        "id",
				Name:      "name",
				Bootstrap: false,
				Expect:    3,
				NonVoter:  false,
				Status:    serf.StatusAlive,
				RPCAddr:   "rpc-addr",
				SerfAddr:  "serf-addr",
			},
			wantOk: true,
		},
		{
			name: "Not Agent Cluster",
			member: serf.Member{
				Tags: map[string]string{
					"cluster":   "something",
					"id":        "id",
					"name":      "name",
					"role":      "agent",
					"expect":    "3",
					"rpc_addr":  "rpc-addr",
					"serf_addr": "serf-addr",
				},
				Status: serf.StatusAlive,
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "Not Agent Role",
			member: serf.Member{
				Tags: map[string]string{
					"cluster":   "test",
					"id":        "id",
					"name":      "name",
					"role":      "something",
					"expect":    "3",
					"rpc_addr":  "rpc-addr",
					"serf_addr": "serf-addr",
				},
				Status: serf.StatusAlive,
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "Invalid Expect",
			member: serf.Member{
				Tags: map[string]string{
					"cluster":   "test",
					"id":        "id",
					"name":      "name",
					"expect":    "foobar",
					"rpc_addr":  "rpc-addr",
					"serf_addr": "serf-addr",
				},
				Status: serf.StatusAlive,
			},
			want:   nil,
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := metadata.IsAgent(tt.member)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantOk, gotOk)
		})
	}
}

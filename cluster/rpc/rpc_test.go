package rpc_test

import (
	"testing"

	"github.com/nrwiersma/cluster/cluster/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadRequest(t *testing.T) {
	tests := []struct {
		name       string
		forceFresh bool
		want       bool
	}{
		{
			name:       "Allow Stale",
			forceFresh: false,
			want:       true,
		},
		{
			name:       "Do Not Allow Stale",
			forceFresh: true,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := rpc.ReadRequest{ForceFresh: tt.forceFresh}

			require.True(t, req.IsRead())
			assert.Equal(t, tt.want, req.AllowStaleRead())
		})
	}
}

func TestWriteRequest(t *testing.T) {
	req := rpc.WriteRequest{}

	require.False(t, req.IsRead())
	assert.False(t, req.AllowStaleRead())
}

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
		allowStale bool
	}{
		{
			name:       "Allow Stale",
			allowStale: true,
		},
		{
			name:       "Do Not Allow Stale",
			allowStale: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := rpc.ReadRequest{AllowStale: tt.allowStale}

			require.True(t, req.IsRead())
			assert.Equal(t, tt.allowStale, req.AllowStaleRead())
		})
	}
}

func TestWriteRequest(t *testing.T) {
	req := rpc.WriteRequest{}

	require.False(t, req.IsRead())
	assert.False(t, req.AllowStaleRead())
}

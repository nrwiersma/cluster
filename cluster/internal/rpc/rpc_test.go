package rpc_test

import (
	"testing"

	"github.com/nrwiersma/cluster/cluster/internal/rpc"
	"github.com/nrwiersma/cluster/cluster/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecode(t *testing.T) {
	req := rpc.RegisterNodeRequest{
		Node: state.Node{
			ID:      "some id",
			Name:    "foobar",
			Role:    "test",
			Address: "addr",
			Health:  "yes",
			Meta: map[string]string{
				"foo": "bar",
			},
		},
	}

	b, err := rpc.Encode(rpc.RegisterNodeRequestType, req)

	require.NoError(t, err)
	assert.Equal(t, byte(rpc.RegisterNodeRequestType), b[0])

	var got rpc.RegisterNodeRequest
	err = rpc.Decode(b[1:], &got)

	require.NoError(t, err)
	assert.Equal(t, req, got)
}

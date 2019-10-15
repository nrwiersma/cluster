package fsm

import (
	"fmt"

	"github.com/hashicorp/go-msgpack/codec"
	"github.com/hashicorp/raft"
	"github.com/nrwiersma/cluster/cluster/internal/rpc"
	"github.com/nrwiersma/cluster/cluster/state"
)

func (f *FSM) handleRegisterNodeRequest(buf []byte, idx uint64) interface{} {
	var req rpc.RegisterNodeRequest
	if err := rpc.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	if err := f.store.EnsureNode(idx, &req.Node); err != nil {
		return err
	}

	return nil
}

func (f *FSM) handleDeregisterNodeRequest(buf []byte, idx uint64) interface{} {
	var req rpc.DeregisterNodeRequest
	if err := rpc.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	if err := f.store.DeleteNode(idx, req.Node.ID); err != nil {
		return err
	}

	return nil
}

func snapshotNodes(s *snapshot, sink raft.SnapshotSink, enc *codec.Encoder) error {
	iter, err := s.state.Nodes()
	if err != nil {
		return err
	}

	for raw := iter.Next(); raw != nil; raw = iter.Next() {
		_, _ = sink.Write([]byte{byte(rpc.RegisterNodeRequestType)})
		if err := enc.Encode(raw.(*state.Node)); err != nil {
			return err
		}
	}
	return nil
}

func restoreNode(header *snapshotHeader, restore *state.Restore, dec *codec.Decoder) error {
	var node state.Node
	if err := dec.Decode(&node); err != nil {
		return err
	}
	return restore.Node(header.LastIndex, &node)
}

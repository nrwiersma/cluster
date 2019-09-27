package fsm

import (
	"fmt"

	"github.com/nrwiersma/cluster/cluster/rpc"
)

func (f *FSM) handleRegisterNode(buf []byte, idx uint64) interface{} {
	var req rpc.RegisterNode
	if err := rpc.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	if err := f.store.EnsureNode(idx, &req.Node); err != nil {
		return err
	}

	return nil
}

func (f *FSM) handleDeregisterNode(buf []byte, idx uint64) interface{} {
	var req rpc.DeregisterNode
	if err := rpc.Decode(buf, &req); err != nil {
		panic(fmt.Errorf("failed to decode request: %v", err))
	}

	if err := f.store.DeleteNode(idx, req.Node.ID); err != nil {
		return err
	}

	return nil
}

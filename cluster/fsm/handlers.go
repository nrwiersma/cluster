package fsm

import (
	"fmt"

	"github.com/nrwiersma/cluster/cluster/rpc"
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

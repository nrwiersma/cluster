package server

import "github.com/nrwiersma/cluster/cluster/rpc"

// Agent serves RPC calls about agents.
type Agent struct {
	srv *Server
}

// GetNodes gets the nodes known to the cluster.
func (a *Agent) GetNodes(req *rpc.NodesRequest, resp *rpc.NodesResponse) error {
	store := a.srv.fsm.Store()

	nodes, err := store.Nodes(nil)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		n := rpc.Node{
			ID:      node.ID,
			Name:    node.Name,
			Role:    node.Role,
			Address: node.Address,
			Health:  string(node.Health),
			Meta:    node.Meta,
		}
		resp.Nodes = append(resp.Nodes, n)
	}

	return nil
}

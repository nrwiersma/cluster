package server

import (
	"github.com/hashicorp/go-bexpr"
	"github.com/nrwiersma/cluster/cluster/rpc"
)

// Cluster serves RPC calls about the cluster.
type Cluster struct {
	srv *Server
}

// GetNodes gets the nodes known to the cluster.
func (a *Cluster) GetNodes(req *rpc.NodesRequest, resp *rpc.NodesResponse) error {
	filter, err := bexpr.CreateFilter(req.Filter, nil, resp.Nodes)
	if err != nil {
		return err
	}

	store := a.srv.fsm().Store()
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

	filtered, err := filter.Execute(resp.Nodes)
	if err != nil {
		return err
	}
	resp.Nodes = filtered.([]rpc.Node)

	return nil
}

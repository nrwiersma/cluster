package server

import (
	"github.com/hashicorp/go-bexpr"
	"github.com/hashicorp/go-memdb"
	"github.com/nrwiersma/cluster/cluster/rpc"
	"github.com/nrwiersma/cluster/cluster/state"
)

// Cluster serves RPC calls about the cluster.
type Cluster struct {
	srv *Server
}

// GetNodes gets the nodes known to the cluster.
func (a *Cluster) GetNodes(req *rpc.NodesRequest, resp *rpc.NodesResponse) error {
	if ok, err := a.srv.state.Forward("Cluster.GetNodes", req, resp); ok {
		return err
	}

	filter, err := bexpr.CreateFilter(req.Filter, nil, resp.Nodes)
	if err != nil {
		return err
	}

	return a.srv.Query(
		&req.ReadRequest,
		&resp.ResponseMeta,
		func(ws memdb.WatchSet, store *state.Store) error {
			idx, nodes, err := store.Nodes(ws)
			if err != nil {
				return err
			}

			resp.Index = idx
			resp.Nodes = nil
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
		},
	)

}

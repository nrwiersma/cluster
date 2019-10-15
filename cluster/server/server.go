package server

import (
	"net/rpc"

	raftrpc "github.com/nrwiersma/cluster/cluster/internal/rpc"
	"github.com/nrwiersma/cluster/cluster/state"
	"github.com/nrwiersma/cluster/pkg/memcodec"
)

// StateDelegate represents an object that can handle state.
type StateDelegate interface {
	Store() *state.Store
	Apply(t raftrpc.MessageType, msg interface{}) (interface{}, error)
}

// Server is an RPC server
type Server struct {
	state StateDelegate

	server *rpc.Server
}

// New returns an RPC server.
func New(state StateDelegate) *Server {
	srv := &Server{
		state:  state,
		server: rpc.NewServer(),
	}

	_ = srv.server.Register(&Cluster{srv: srv})

	return srv
}

// ServeRequest serves a single request with the given codec.
func (s *Server) ServeRequest(codec rpc.ServerCodec) error {
	return s.server.ServeRequest(codec)
}

// Call makes an in memory call to the server.
func (s *Server) Call(method string, req, resp interface{}) error {
	codec := memcodec.New(method, req, resp)
	if err := s.server.ServeRequest(codec); err != nil {
		return err
	}
	return codec.Error
}

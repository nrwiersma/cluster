package server

import (
	"net/rpc"

	"github.com/nrwiersma/cluster/cluster/internal/fsm"
	raftrpc "github.com/nrwiersma/cluster/cluster/internal/rpc"
)

type Applier func(t raftrpc.MessageType, msg interface{}) (interface{}, error)

// Server is an RPC server
type Server struct {
	fsm   *fsm.FSM
	apply Applier

	server *rpc.Server
}

// New returns an RPC server.
func New(fsm *fsm.FSM, apply Applier) *Server {
	srv := &Server{
		fsm:    fsm,
		apply:  apply,
		server: rpc.NewServer(),
	}

	_ = srv.server.Register(&Agent{srv: srv})

	return srv
}

// ServerRequest serves a single request with the given codec.
func (s *Server) ServeRequest(codec rpc.ServerCodec) error {
	return s.server.ServeRequest(codec)
}

// Call makes an in memory call to the server.
func (s *Server) Call(method string, req, resp interface{}) error {
	// TODO: make an in memory server call

	return nil
}

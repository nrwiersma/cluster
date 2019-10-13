package server

import (
	"net/rpc"

	"github.com/nrwiersma/cluster/cluster/internal/fsm"
	raftrpc "github.com/nrwiersma/cluster/cluster/internal/rpc"
	"github.com/nrwiersma/cluster/pkg/memcodec"
)

// FSM represents a function that returns an FSM.
type FSM func() *fsm.FSM

// Apply represents a function that can apply to the raft state.
type Apply func(t raftrpc.MessageType, msg interface{}) (interface{}, error)

// Server is an RPC server
type Server struct {
	fsm   FSM
	apply Apply

	server *rpc.Server
}

// New returns an RPC server.
func New(fsm FSM, apply Apply) *Server {
	srv := &Server{
		fsm:    fsm,
		apply:  apply,
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

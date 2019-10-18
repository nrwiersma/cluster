package server

import (
	"net/rpc"
	"time"

	"github.com/hashicorp/go-memdb"
	raftrpc "github.com/nrwiersma/cluster/cluster/internal/rpc"
	rpc2 "github.com/nrwiersma/cluster/cluster/rpc"
	"github.com/nrwiersma/cluster/cluster/state"
	"github.com/nrwiersma/cluster/pkg/memcodec"
)

const (
	maxQueryTime     = 600 * time.Second
	defaultQueryTime = 300 * time.Second
)

// StateDelegate represents an object that can handle state.
type StateDelegate interface {
	Store() *state.Store
	Forward(string, interface{}, interface{}) (bool, error)
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

type queryFn func(memdb.WatchSet, *state.Store) error

func (s *Server) Query(req *rpc2.ReadRequest, meta *rpc2.ResponseMeta, fn queryFn) error {
	var timeout *time.Timer

	if req.MinQueryIndex > 0 {
		queryTimeout := req.MaxQueryTime
		if queryTimeout > maxQueryTime {
			queryTimeout = maxQueryTime
		} else if queryTimeout <= 0 {
			queryTimeout = defaultQueryTime
		}

		timeout = time.NewTimer(queryTimeout)
		defer timeout.Stop()
	}

	var err error
	for {
		store := s.state.Store()

		var ws memdb.WatchSet
		if req.MinQueryIndex > 0 {
			ws = memdb.NewWatchSet()

			ws.Add(store.AbandonCh())
		}

		err = fn(ws, store)

		if err == nil && meta.Index < 1 {
			meta.Index = 1
		}

		if err == nil && req.MinQueryIndex > 0 && meta.Index <= req.MinQueryIndex {
			if expired := ws.Watch(timeout.C); !expired {
				continue
			}
		}

		return err
	}
}

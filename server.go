package sockerball

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shabbyrobe/sockerball/internal/incrementer"
)

type (
	ServerOption       func(srv *Server)
	OnServerConnect    func(server *Server, id ConnID)
	OnServerDisconnect func(server *Server, id ConnID, err error)
	OnServerError      func(server *Server, id ConnID, err error)
)

// ServerConnect is a ServerOption that registers a callback that happens when
// a client connects to the server.
func ServerConnect(cb OnServerConnect) ServerOption {
	return func(srv *Server) { srv.onConnect = cb }
}

// ServerDisconnect is a ServerOption that registers a callback that happens when
// a client disconnects from the server.
func ServerDisconnect(cb OnServerDisconnect) ServerOption {
	return func(srv *Server) { srv.onDisconnect = cb }
}

// ServerError registers a function to be called when a non-fatal error
// occurs within the server
func ServerError(cb OnServerError) ServerOption {
	return func(srv *Server) { srv.onError = cb }
}

type Server struct {
	config       ServerConfig
	handler      Handler
	negotiator   Negotiator
	onConnect    OnServerConnect
	onDisconnect OnServerDisconnect
	onError      OnServerError

	nextID incrementer.Inc

	conns   map[ConnID]*conn
	connsMu sync.Mutex
	running uint32
}

func NewServer(config *ServerConfig, negotiator Negotiator, handler Handler, opts ...ServerOption) *Server {
	if handler == nil {
		panic("socket: handler must not be nil")
	}
	if negotiator == nil {
		panic("socket: negotiator must not be nil")
	}
	if config == nil || config.IsZero() {
		config = DefaultServerConfig()
	}

	srv := &Server{
		config:     *config,
		conns:      make(map[ConnID]*conn),
		handler:    handler,
		negotiator: negotiator,
	}
	for _, o := range opts {
		o(srv)
	}

	return srv
}

func (srv *Server) onEnd(id ConnID, err error) {
	srv.connsMu.Lock()
	delete(srv.conns, id)
	srv.connsMu.Unlock()
	if srv.onDisconnect != nil {
		srv.onDisconnect(srv, id, err)
	}
}

func (srv *Server) Serve(listener Listener) (rerr error) {
	if !atomic.CompareAndSwapUint32(&srv.running, 0, 1) {
		return fmt.Errorf("socket: server already running")
	}

	defer func() {
		if cerr := listener.Close(); cerr != nil && rerr == nil {
			rerr = cerr
		}
		atomic.StoreUint32(&srv.running, 0)
	}()

	for {
		raw, err := listener.Accept()
		if err != nil {
			const delay = 100 * time.Millisecond // FIXME: backoff
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				// FIXME: log
				time.Sleep(delay)
				continue
			} else {
				return err
			}
		}

		id := ConnID(srv.nextID.Next())

		go srv.runConn(id, raw)
	}
}

func (srv *Server) runConn(id ConnID, raw Communicator) {
	conn := newConn(id, ServerSide, srv.config.Conn, raw, srv.negotiator, srv.handler)

	// we must start the service and raise the onConnected event
	// while the lock is acquired otherwise the "onDisconnect"
	// callback can be called before the "onConnect" callback.
	srv.connsMu.Lock()
	defer srv.connsMu.Unlock()

	clientEnded := make(chan error, 1)
	if err := conn.start(clientEnded); err != nil {
		srv.onError(srv, id, err)
		return
	}

	go func() {
		srv.onEnd(id, <-clientEnded)
	}()

	if srv.onConnect != nil {
		srv.onConnect(srv, id)
	}
	srv.conns[id] = conn
}

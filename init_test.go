package sockerball

import (
	"context"
	"testing"
	"time"

	"github.com/shabbyrobe/golib/assert"
)

type testingListener struct {
	communicators chan Communicator
}

var _ Listener = &testingListener{}

func newTestingListener() *testingListener {
	return &testingListener{
		communicators: make(chan Communicator, 1),
	}
}

func (tl *testingListener) Accept() (Communicator, error) {
	return <-tl.communicators, nil
}

func (tl *testingListener) Close() error {
	return nil
}

type testingCommunicator struct {
	reader chan []byte
	writer chan []byte
	ping   []byte
	close  chan struct{}
}

var _ Communicator = &testingCommunicator{}

func newTestingCommunicator(buffer int) *testingCommunicator {
	return &testingCommunicator{
		reader: make(chan []byte, buffer),
		writer: make(chan []byte, buffer),
		ping:   []byte{0},
		close:  make(chan struct{}),
	}
}

func (tc *testingCommunicator) Close() error {
	close(tc.close)
	return nil
}

func (tc *testingCommunicator) Ping(timeout time.Duration) error {
	return nil
}

func (tc *testingCommunicator) Pongs() <-chan struct{} {
	return nil
}

func (tc *testingCommunicator) MessageLimit() int {
	panic("not implemented")
}

func (tc *testingCommunicator) ReadMessage(into []byte, limit int, timeout time.Duration) (extended []byte, rerr error) {
	panic("not implemented")
}

func (tc *testingCommunicator) WriteMessage(data []byte, timeout time.Duration) (rerr error) {
	panic("not implemented")
}

type testingServer struct {
	config          *ServerConfig
	server          *Server
	handler         Handler
	negotiator      Negotiator
	protocol        Protocol
	testingListener *testingListener
	serverOpts      []ServerOption
}

func newTestingServer(tt assert.T, opts ...serverOpt) *testingServer {
	tt.Helper()
	ts := &testingServer{
		config: DefaultServerConfig(),
	}
	ts.handler = ts
	for _, o := range opts {
		o(ts)
	}
	if ts.negotiator == nil {
		ts.negotiator = ts
	}
	if ts.protocol == nil {
		ts.protocol = ts
	}

	ts.server = NewServer(ts.config, ts.negotiator, ts.handler, ts.serverOpts...)
	return ts
}

func (ts *testingServer) Listen(tt assert.T, l Listener) {
	if l == nil {
		l = newTestingListener()
	}
	tt.MustOK(ts.server.Serve(l))
}

func (ts *testingServer) Negotiate(Side, Communicator, ConnConfig) (Protocol, error) {
	return ts.protocol, nil
}

func (ts *testingServer) HandleRequest(ctx context.Context, ir IncomingRequest) (rs Message, rerr error) {
	return nil, nil
}

func (ts *testingServer) Mapper() Mapper {
	panic("not implemented")
}

func (ts *testingServer) MessageLimit() int {
	panic("not implemented")
}

func (ts *testingServer) ProtocolName() string {
	panic("not implemented")
}

func (ts *testingServer) Codec() Codec {
	panic("not implemented")
}

type testingProtocol struct{}

func (ts *testingProtocol) Mapper() Mapper {
	panic("not implemented")
}

func (ts *testingProtocol) MessageLimit() int {
	panic("not implemented")
}

func (ts *testingProtocol) ProtocolName() string {
	panic("not implemented")
}

func (ts *testingProtocol) Codec() Codec {
	panic("not implemented")
}

type serverOpt func(s *testingServer)

func TestServer(t *testing.T) {
	tt := assert.WrapTB(t)
	_ = tt

}

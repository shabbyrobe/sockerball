package packetsrv

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/shabbyrobe/sockerball"
)

const DefaultMessageLimit int = 508

type communicator struct {
	addr         net.Addr
	reader       chan readMsg
	writer       chan<- writeMsg
	closer       chan<- net.Addr
	stop         chan struct{}
	messageLimit int

	// lastRead is owned for reading and writing by the Listener, not by this
	// struct.
	lastRead time.Time
}

var _ sockerball.Communicator = &communicator{}

func newCommunicator(
	addr net.Addr,
	reader chan readMsg,
	writer chan<- writeMsg,
	closer chan<- net.Addr,
	stop chan struct{},
) *communicator {

	return &communicator{
		addr:   addr,
		reader: reader,
		writer: writer,
		closer: closer,
		stop:   stop,

		// Thus, safe packet size of 508 = 576 - 60 (IP header) - 8 (udp header) is reasonable.
		// FIXME: configurable
		messageLimit: DefaultMessageLimit,
	}
}

func (pc *communicator) MessageLimit() int {
	return pc.messageLimit
}

func (pc *communicator) Close() error {
	select {
	case pc.closer <- pc.addr:
	case <-pc.stop:
	}
	return nil
}

func (pc *communicator) ReadMessage(into []byte, limit int, timeout time.Duration) (buf []byte, rerr error) {
	// FIXME: need some sort of sync.Cond-based shitshow for efficiency here,
	// this is foul.
	tc := time.After(timeout)
	select {
	case msg, ok := <-pc.reader:
		if ok {
			return msg.buf, nil
		} else {
			return into, io.EOF
		}
	case <-pc.stop:
		return nil, io.EOF
	case <-tc:
		return into, fmt.Errorf("packetsrv: timeout")
	}
}

func (pc *communicator) WriteMessage(data []byte, timeout time.Duration) (rerr error) {
	// FIXME: need some sort of sync.Cond-based shitshow for efficiency here,
	// this is foul.
	tc := time.After(timeout)
	errc := make(chan error, 1)

	select {
	case pc.writer <- writeMsg{addr: pc.addr, buf: data, errc: errc}:
		return <-errc
	case <-pc.stop:
		return io.EOF
	case <-tc:
		return fmt.Errorf("packetsrv: write timeout")
	}
}

func (pc *communicator) Ping(timeout time.Duration) (rerr error) {
	return pc.WriteMessage(packetPingBuf, timeout)
}

func (pc *communicator) Pongs() <-chan struct{} {
	return nil
}

var packetPingBuf = []byte{0}

const packetPingBufLen = 1

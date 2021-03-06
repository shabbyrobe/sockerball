package packetsrv

import (
	"fmt"
	"net"
	"time"

	"github.com/shabbyrobe/sockerball"
)

type clientCommunicator struct {
	conn         net.Conn
	messageLimit int
}

var _ sockerball.Communicator = &communicator{}

func ClientCommunicator(conn net.Conn) sockerball.Communicator {
	return &clientCommunicator{
		conn:         conn,
		messageLimit: DefaultMessageLimit,
	}
}

func (pc *clientCommunicator) MessageLimit() int {
	return pc.messageLimit
}

func (pc *clientCommunicator) Close() error {
	return pc.conn.Close()
}

func (pc *clientCommunicator) ReadMessage(into []byte, limit int, timeout time.Duration) (buf []byte, rerr error) {
	if timeout > 0 {
		if err := pc.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			return into, err
		}
	}

	if cap(into) < limit {
		into = make([]byte, limit)
	} else {
		into = into[:limit]
	}

	n, err := pc.conn.Read(into)
	if err != nil {
		return into, err
	}

	return into[:n], nil
}

func (pc *clientCommunicator) WriteMessage(data []byte, timeout time.Duration) (rerr error) {
	if timeout > 0 {
		if err := pc.conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
			return err
		}
	}

	if n, err := pc.conn.Write(data); err != nil {
		return err
	} else if n != len(data) {
		return fmt.Errorf("short message write")
	}

	return nil
}

func (pc *clientCommunicator) Ping(timeout time.Duration) (rerr error) {
	return pc.WriteMessage(packetPingBuf, timeout)
}

func (pc *clientCommunicator) Pongs() <-chan struct{} {
	return nil
}

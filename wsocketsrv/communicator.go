package wsocketsrv

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shabbyrobe/sockerball"
)

type Communicator struct {
	ws       *websocket.Conn
	pongs    chan struct{}
	rdLenBuf [4]byte
	wrLenBuf [4]byte
}

var _ sockerball.Communicator = &Communicator{}

func NewCommunicator(ws *websocket.Conn) *Communicator {
	comm := &Communicator{
		ws:    ws,
		pongs: make(chan struct{}, 1),
	}

	existing := ws.PongHandler()
	ws.SetPongHandler(func(s string) error {
		select {
		case comm.pongs <- struct{}{}:
		default:
		}
		if existing != nil {
			return existing(s)
		}
		return nil
	})

	return comm
}

func (cm *Communicator) MessageLimit() int { return 0 }

func (cm *Communicator) Close() error {
	return cm.ws.Close()
}

func (cm *Communicator) ReadMessage(into []byte, limit int, timeout time.Duration) (extended []byte, rerr error) {
	// NextReader does not receive pongs, we are using the websocket's
	// SetPongHandler for that job, so we can't set a read timeout here as the
	// pongs will never be read this way and the timeout will occur even
	// when the heartbeats are sent.

	_, rdr, err := cm.ws.NextReader()
	if err != nil {
		return nil, err
	}

	lbuf := cm.rdLenBuf[:]
	if _, err := io.ReadFull(rdr, lbuf); err != nil {
		return into, err
	}

	// The websocket protocol makes length available as part of the header,
	// but the gorilla library does not expose the field for us to validate:
	mlen := int(binary.BigEndian.Uint32(lbuf))
	if mlen > limit {
		return into, fmt.Errorf("socket: message of length %d exceeded limit %d", mlen, uint32(limit))
	}

	if cap(into) < mlen {
		into = make([]byte, mlen)
	} else {
		into = into[:mlen]
	}

	if _, err := io.ReadFull(rdr, into); err != nil {
		return into, err
	}

	return into, nil
}

func (cm *Communicator) WriteMessage(data []byte, timeout time.Duration) (rerr error) {
	if timeout > 0 {
		if err := cm.ws.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
			return err
		}
	}

	mlen := len(data)
	wr, err := cm.ws.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}

	lbuf := cm.wrLenBuf[:]
	binary.BigEndian.PutUint32(lbuf, uint32(mlen))
	if n, err := wr.Write(lbuf); err != nil {
		_ = wr.Close()
		return err

	} else if n != 4 {
		_ = wr.Close()
		return fmt.Errorf("short length write")
	}

	if n, err := wr.Write(data); err != nil {
		_ = wr.Close()
		return err
	} else if n != mlen {
		_ = wr.Close()
		return fmt.Errorf("short message write")
	}

	return wr.Close()
}

func (cm *Communicator) Ping(timeout time.Duration) (rerr error) {
	if timeout > 0 {
		if err := cm.ws.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
			return err
		}
	}

	return cm.ws.WritePreparedMessage(ping)
}

func (cm *Communicator) Pongs() <-chan struct{} {
	return cm.pongs
}

package packetsrv

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/shabbyrobe/sockerball"
	"github.com/shabbyrobe/sockerball/internal/incrementer"
)

func Listen(network, addr string) (sockerball.Listener, error) {
	pl := &listener{
		network: network,
		addr:    addr,
		accept:  make(chan *communicator, 1024), // FIXME: buffer size
		stop:    make(chan struct{}),
	}
	go func() {
		pl.listen()
	}()
	return pl, nil
}

type listener struct {
	network string
	addr    string
	inc     incrementer.Inc
	accept  chan *communicator

	stop chan struct{}
}

func (pl *listener) listen() error {
	pc, err := net.ListenPacket(pl.network, pl.addr)
	if err != nil {
		return err
	}

	var (
		errc   = make(chan error, 3)
		reader = make(chan readMsg, 1024)
		writer = make(chan writeMsg, 1024)
		closer = make(chan net.Addr, 1024)
		comms  = make(map[string]*communicator)
	)

	commClose := func(key string, comm *communicator) {
		close(comm.reader)
		close(comm.stop)
		delete(comms, key)
	}

	var wg sync.WaitGroup
	wg.Add(2) // reader and writer

	go func() { // reader thread
		defer wg.Done()

		into := make([]byte, 65536)
		for {
			// FIXME: SetReadDeadline()
			n, addr, err := pc.ReadFrom(into)
			if err != nil {
				errc <- err
				return
			}

			buf := make([]byte, n)
			copy(buf, into)

			select {
			case reader <- readMsg{n, addr, buf}:
			case <-pl.stop:
				return
			}
		}
	}()

	go func() { // writer thread
		defer wg.Done()

		for {
			select {
			case out := <-writer:
				_, err := pc.WriteTo(out.buf, out.addr)
				// errc MUST have a one-element buffer.
				out.errc <- err

			case <-pl.stop:
				return
			}
		}
	}()

	defer func() {
		_ = pc.Close()
		wg.Wait()

		for addr, comm := range comms {
			commClose(addr, comm)
		}
	}()

	// FIXME: configurable
	cleanup := time.NewTicker(1 * time.Second)
	defer cleanup.Stop()

	for {
		select {
		case at := <-cleanup.C:
			for addr, comm := range comms {
				if at.Sub(comm.lastRead) > 5*time.Second {
					commClose(addr, comm)
				}
			}

		case err := <-errc:
			return err

		case addr := <-closer:
			akey := addr.String()
			comm := comms[akey]
			if comm != nil {
				commClose(akey, comm)
			}

		case msg := <-reader:
			akey := msg.addr.String()
			comm, ok := comms[akey]
			if !ok {
				commReader := make(chan readMsg, 1024) // FIXME: configurable
				stop := make(chan struct{})
				comm = newCommunicator(msg.addr, commReader, writer, closer, stop)
				comms[akey] = comm

				select {
				case pl.accept <- comm:
				default:
					// FIXME: maybe a timeout?
					return fmt.Errorf("packetsrv: accept buffer full")
				}
			}

			select {
			case comm.reader <- msg:
				comm.lastRead = time.Now()
			default:
				// FIXME: kill the connection
			}

		case <-pl.stop:
			return nil
		}
	}

	return nil
}

func (pl *listener) Accept() (sockerball.Communicator, error) {
	select {
	case comm := <-pl.accept:
		return comm, nil
	case <-pl.stop:
		return nil, errors.New("packetsrv: listener closed")
	}
}

func (pl *listener) Close() (rerr error) {
	defer func() {
		if err := recover(); err != nil {
			rerr = fmt.Errorf("packetsrv: listener already closed")
		}
	}()
	close(pl.stop)
	return nil
}

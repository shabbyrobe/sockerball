package sockerball

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Negotiator negotiates the Protocol used by the client and the server.
// Negotiation should perform at least one write and read using the
// Communicator. Be careful to ensure that your negotiator doesn't limit your
// ability to upgrade the protocol in the future; that's the whole point of it.
//
// See NewVersionNegotiator for a quick way to include support for version
// negotiation.
type Negotiator interface {
	Negotiate(Side, Communicator, ConnConfig) (Protocol, error)
}

type Mapper interface {
	Message(kind int) (Message, error)
	MessageKind(msg Message) (int, error)
}

type Protocol interface {
	Mapper() Mapper

	MessageLimit() int
	ProtocolName() string

	// Codec is cached by Conn
	Codec() Codec
}

type Codec interface {
	Decode(in []byte, mapper Mapper, decdata *ProtoData) (Envelope, error)
	Encode(env Envelope, into []byte, encdata *ProtoData) (extended []byte, rerr error)
}

// ProtoData is used as a type-unsafe way for a Protocol to store shared memory
// against a Conn object for reuse.
type ProtoData interface {
	io.Closer
}

type VersionedProtocol interface {
	Protocol
	Version() int
}

type VersionNegotiator struct {
	protocols map[uint32]VersionedProtocol
	encoding  binary.ByteOrder
	ours      []byte
}

var _ Negotiator = &VersionNegotiator{}

func NewVersionNegotiator(protos ...VersionedProtocol) *VersionNegotiator {
	if len(protos) == 0 {
		panic("socketsrv: no procols specified")
	}
	vn := &VersionNegotiator{
		protocols: make(map[uint32]VersionedProtocol),
		encoding:  binary.BigEndian,
	}

	var ours = make([]byte, len(protos)*4)
	for i, p := range protos {
		if p.Version() < 0 || int64(p.Version()) > int64(1<<32-1) {
			panic("socketsrv: proto version must fit inside uint32")
		}
		pv := uint32(p.Version())
		if vn.protocols[pv] != nil {
			panic(fmt.Errorf("socketsrv: duplicate proto version %d", pv))
		}
		vn.encoding.PutUint32(ours[i*4:], pv)
		vn.protocols[pv] = p
	}
	vn.ours = ours

	return vn
}

// Limit returns a copy of the VersionNegotiator, limited to the versions
// passed.
func (v *VersionNegotiator) Limit(versions ...int) (*VersionNegotiator, error) {
	protos := make([]VersionedProtocol, len(versions))
	for i, ver := range versions {
		p := v.protocols[uint32(ver)]
		if p == nil {
			return nil, fmt.Errorf("socketsrv: could not find version %d", ver)
		}
		protos[i] = p
	}
	return NewVersionNegotiator(protos...), nil
}

func (v *VersionNegotiator) Negotiate(side Side, c Communicator, config ConnConfig) (Protocol, error) {
	if err := c.WriteMessage(v.ours, config.WriteTimeout); err != nil {
		return nil, err
	}

	msg, err := c.ReadMessage(nil, 1024, config.ReadTimeout)
	if err != nil {
		return nil, err
	}
	if len(msg)%4 != 0 {
		return nil, fmt.Errorf("unexpected remote versions")
	}

	var max uint32
	var found bool
	for i := 0; i < len(msg); i += 4 {
		cur := v.encoding.Uint32(msg[i:])
		if _, ok := v.protocols[cur]; ok {
			found = true
			if cur > max {
				max = cur
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("could not negotiate protocol")
	}

	return v.protocols[max], nil
}

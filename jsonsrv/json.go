package jsonsrv

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/shabbyrobe/sockerball"
)

var Codec = NewCodec()

type codec struct {
	encoding binary.ByteOrder
}

type Option func(jp *codec)

func Encoding(bo binary.ByteOrder) Option {
	return func(jp *codec) { jp.encoding = bo }
}

func NewCodec(opts ...Option) sockerball.Codec {
	jp := &codec{}
	for _, o := range opts {
		o(jp)
	}
	if jp.encoding == nil {
		jp.encoding = binary.LittleEndian
	}
	return jp
}

func (p *codec) Decode(in []byte, mapper sockerball.Mapper, decdata *sockerball.ProtoData) (env sockerball.Envelope, err error) {
	if len(in) < 12 {
		return env, fmt.Errorf("socketsrv: short message")
	}

	env.ID = sockerball.MessageID(p.encoding.Uint32(in))
	env.ReplyTo = sockerball.MessageID(p.encoding.Uint32(in[4:]))
	env.Kind = int(p.encoding.Uint32(in[8:]))
	env.Message, err = mapper.Message(env.Kind)
	if err != nil {
		return env, err
	}

	if err := json.Unmarshal(in[12:], &env.Message); err != nil {
		return env, err
	}

	return env, nil
}

func (p *codec) Encode(env sockerball.Envelope, into []byte, encData *sockerball.ProtoData) (extended []byte, rerr error) {
	var hdr [12]byte
	p.encoding.PutUint32(hdr[0:], uint32(env.ID))
	p.encoding.PutUint32(hdr[4:], uint32(env.ReplyTo))
	p.encoding.PutUint32(hdr[8:], uint32(env.Kind))

	buf := bytes.NewBuffer(into[:0])
	buf.Write(hdr[:])

	enc := json.NewEncoder(buf)
	if err := enc.Encode(env.Message); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

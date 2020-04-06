package exampleproto

import (
	"encoding/json"

	"github.com/shabbyrobe/golib/iotools/bytewriter"
	"github.com/shabbyrobe/sockerball"
)

type SimpleCodec struct{}

var _ sockerball.Codec = SimpleCodec{}

func (p SimpleCodec) Decode(in []byte, mapper sockerball.Mapper, decdata *sockerball.ProtoData) (env sockerball.Envelope, rerr error) {
	var je JSONEnvelope
	if err := json.Unmarshal(in, &je); err != nil {
		return env, err
	}

	msg, err := mapper.Message(je.Kind)
	if err != nil {
		return env, err
	}

	if err := json.Unmarshal(je.Message, &msg); err != nil {
		return env, err
	}

	env.ID = je.ID
	env.ReplyTo = je.ReplyTo
	env.Kind = je.Kind
	env.Message = msg
	return env, nil
}

func (p SimpleCodec) Encode(env sockerball.Envelope, into []byte, encdata *sockerball.ProtoData) (extended []byte, rerr error) {
	var bw bytewriter.Writer
	bw.Give(into[:0])
	enc := json.NewEncoder(&bw)
	if err := enc.Encode(env.Message); err != nil {
		return nil, err
	}

	raw, n := bw.Take()
	je := JSONEnvelope{
		ID:      env.ID,
		ReplyTo: env.ReplyTo,
		Kind:    env.Kind,
		Message: raw[:n],
	}

	bw.Give(raw)
	enc = json.NewEncoder(&bw)
	if err := enc.Encode(je); err != nil {
		return nil, err
	}

	out, ilen := bw.Take()
	copy(out, out[n:])
	out = out[:ilen-n]

	return out, nil
}

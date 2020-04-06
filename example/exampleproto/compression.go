package exampleproto

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/shabbyrobe/golib/iotools/bytewriter"
	"github.com/shabbyrobe/sockerball"
)

var encoding = binary.LittleEndian

type CompressingCodec struct{}

var _ sockerball.Codec = &CompressingCodec{}

func (p CompressingCodec) Decode(in []byte, mapper sockerball.Mapper, decData *sockerball.ProtoData) (env sockerball.Envelope, rerr error) {
	var data *altProtoData
	if *decData == nil {
		data = &altProtoData{}
		*decData = data
	} else {
		data = (*decData).(*altProtoData)
	}

	if len(in) < 13 {
		return env, fmt.Errorf("short message")
	}
	env.ID = sockerball.MessageID(encoding.Uint32(in))
	env.ReplyTo = sockerball.MessageID(encoding.Uint32(in[4:]))
	env.Kind = int(encoding.Uint32(in[8:]))
	env.Message, rerr = mapper.Message(env.Kind)

	compressed := in[12]&0x1 == 0x1
	msg := in[13:]
	if compressed {
		dr := flate.NewReader(bytes.NewReader(msg))
		var bw bytewriter.Writer
		var n int
		bw.Give(data.scratch[:0])
		if _, err := io.Copy(&bw, dr); err != nil {
			return env, err
		}
		msg, n = bw.Take()
		msg = msg[:n]
	}

	if err := json.Unmarshal(msg, &env.Message); err != nil {
		return env, err
	}

	return env, nil
}

func (p CompressingCodec) Encode(env sockerball.Envelope, into []byte, encData *sockerball.ProtoData) (extended []byte, rerr error) {
	var data *altProtoData
	if *encData == nil {
		fw, err := flate.NewWriter(ioutil.Discard, 9)
		if err != nil {
			return into, err
		}
		data = &altProtoData{
			flateWriter: fw,
		}
		*encData = data
	} else {
		data = (*encData).(*altProtoData)
	}

	if len(into) < 13 {
		into = make([]byte, 13)
	}
	encoding.PutUint32(into, uint32(env.ID))
	encoding.PutUint32(into[4:], uint32(env.ReplyTo))
	encoding.PutUint32(into[8:], uint32(env.Kind))
	into[12] = 0

	var bw bytewriter.Writer
	bw.Give(into[:13])

	enc := json.NewEncoder(&bw)
	if err := enc.Encode(env.Message); err != nil {
		return into, err
	}

	out, ilen := bw.Take()

	if ilen >= 5000 {
		into[12] |= 0x01

		if len(data.scratch) < 13 {
			data.scratch = make([]byte, 13)
		}
		copy(data.scratch, into[:13])

		bw.Give(data.scratch[:13])
		cw := data.flateWriter
		cw.Reset(&bw)
		if w, err := cw.Write(out[13:ilen]); err != nil {
			_ = cw.Close()
			return into, err
		} else if w != ilen-13 {
			_ = cw.Close()
			return into, fmt.Errorf("short write")
		}
		if err := cw.Flush(); err != nil {
			_ = cw.Close()
			return into, err
		}
		if err := cw.Close(); err != nil {
			_ = cw.Close()
			return into, err
		}

		out, ilen = bw.Take()
	}
	return out[:ilen], nil
}

type altProtoData struct {
	scratch     []byte
	flateWriter *flate.Writer
}

func (a *altProtoData) Close() error {
	return nil
}

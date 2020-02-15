package main

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/binary"
	"encoding/json"
	"expvar"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/websocket"
	"github.com/pkg/profile"
	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/arg"
	service "github.com/shabbyrobe/go-service"
	"github.com/shabbyrobe/go-service/services"
	"github.com/shabbyrobe/go-service/serviceutil"
	"github.com/shabbyrobe/golib/iotools/bytewriter"
	"github.com/shabbyrobe/sockerball"
	"github.com/shabbyrobe/sockerball/jsonsrv"
	"github.com/shabbyrobe/sockerball/packetsrv"
	"github.com/shabbyrobe/sockerball/wsocketsrv"
)

var encoding = binary.BigEndian

func main() {
	if err := run(); err != nil {
		cmdy.Fatal(err)
	}
}

func run() error {
	bld := func() cmdy.Command {
		return cmdy.NewGroup("socketjunk", cmdy.Builders{
			"tcpclient": func() cmdy.Command { return &tcpClientCommand{} },
			"tcpserver": func() cmdy.Command { return &tcpServerCommand{} },
			"pktclient": func() cmdy.Command { return &pktClientCommand{} },
			"pktserver": func() cmdy.Command { return &pktServerCommand{} },
			"wsclient":  func() cmdy.Command { return &wsClientCommand{} },
			"wsserver":  func() cmdy.Command { return &wsServerCommand{} },
		})
	}

	return cmdy.Run(context.Background(), os.Args[1:], bld)
}

func negotiatorBuild() sockerball.Negotiator {
	negotiator := sockerball.NewVersionNegotiator(
		Proto{version: 1, messageLimit: 1000000, mapper: Mapper{}, codec: SimpleCodec{}},
		Proto{version: 2, messageLimit: 1000000, mapper: Mapper{}, codec: jsonsrv.Codec},
		// CompressingProto{},
	)
	return negotiator
}

type tcpClientCommand struct {
	host    string
	spammer spammer
}

func (cl *tcpClientCommand) Help() cmdy.Help { return cmdy.Synopsis("tcpclient") }

func (cl *tcpClientCommand) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	cl.spammer.Flags(flags)
	args.StringOptional(&cl.host, "host", "localhost:9631", "host")
}

func (cl *tcpClientCommand) Run(ctx cmdy.Context) error {
	// defer profile.Start().Stop()
	debugServer(":14440")

	dialer := cl.spammer.Dialer(negotiatorBuild())

	clientCb := func(handler sockerball.Handler, opts ...sockerball.ClientOption) (sockerball.Client, error) {
		client, err := dialer.DialStream(ctx, "tcp", cl.host, handler, opts...)
		if err != nil {
			return nil, err
		}
		return client, nil
	}
	return cl.spammer.Spam(ctx, nil, clientCb)
}

type tcpServerCommand struct {
	host string
}

func (sc *tcpServerCommand) Help() cmdy.Help { return cmdy.Synopsis("tcpserver") }

func (sc *tcpServerCommand) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	flags.StringVar(&sc.host, "host", ":9631", "host")
}

func (sc *tcpServerCommand) Run(ctx cmdy.Context) error {
	// defer profile.Start(profile.BlockProfile).Stop()

	debugServer(":14441")

	handler := &ServerHandler{}

	var config = sockerball.DefaultServerConfig()
	config.Conn.ResponseTimeout = 20 * time.Second
	config.Conn.ReadTimeout = 20 * time.Second
	config.Conn.WriteTimeout = 20 * time.Second

	ln, err := sockerball.ListenStream("tcp", sc.host)
	if err != nil {
		return err
	}

	srv := sockerball.NewServer(config, ln, negotiatorBuild(), handler,
		sockerball.ServerConnect(func(srv *sockerball.Server, id sockerball.ConnID) {
			fmt.Println("connect", id)
		}),
		sockerball.ServerDisconnect(func(srv *sockerball.Server, id sockerball.ConnID, err error) {
			fmt.Println("disconnect", id, err)
		}),
	)

	go srv.Serve()

	fmt.Printf("listening on %s\n", sc.host)

	select {}
}

type pktClientCommand struct {
	host string
}

func (cl *pktClientCommand) Help() cmdy.Help { return cmdy.Synopsis("pktclient") }

func (cl *pktClientCommand) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	args.StringOptional(&cl.host, "host", "localhost:9633", "host")
}

func (cl *pktClientCommand) Run(ctx cmdy.Context) error {
	socketDialer := sockerball.DefaultDialer(negotiatorBuild())

	dialer := net.Dialer{}
	handler := &ServerHandler{}

	in, err := ioutil.ReadFile("/Users/bl/Downloads/The Rust Programming Language.htm")
	if err != nil {
		return err
	}
	in = in[:50]
	_ = in

	s := time.Now()

	iter := 200
	threads := 1000
	var wg sync.WaitGroup
	wg.Add(threads)

	for thread := 0; thread < threads; thread++ {
		go func(thread int) {
			defer wg.Done()

			conn, err := dialer.DialContext(ctx, "udp", cl.host)
			if err != nil {
				panic(err)
			}
			client, err := socketDialer.Client(ctx, packetsrv.ClientCommunicator(conn), handler)
			if err != nil {
				panic(err)
			}

			rq := &TestRequest{
				Foo: fmt.Sprintf("%d", thread),
			}
			for i := 0; i < iter; i++ {
				rsp, err := (client.Request(ctx, rq))
				if err != nil {
					panic(err)
				}
				_ = rsp
				// fmt.Printf("%#v\n", rsp)
			}
		}(thread)
	}

	wg.Wait()
	spew.Dump(time.Since(s))

	return nil
}

type pktServerCommand struct {
	host string
}

func (sc *pktServerCommand) Help() cmdy.Help { return cmdy.Synopsis("pktserver") }

func (sc *pktServerCommand) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	flags.StringVar(&sc.host, "host", ":9633", "host")
}

func (sc *pktServerCommand) Run(ctx cmdy.Context) error {
	defer profile.Start().Stop()

	debugServer(":14442")

	handler := &ServerHandler{}

	ln, err := packetsrv.Listen("udp", sc.host)
	if err != nil {
		return err
	}

	srv := sockerball.NewServer(nil, ln, negotiatorBuild(), handler)
	ender := service.NewEndListener(1)
	svc := service.New(service.Name(sc.host), srv).WithEndListener(ender)
	if err := service.StartTimeout(5*time.Second, services.Runner(), svc); err != nil {
		return err
	}
	fmt.Printf("listening on %s\n", sc.host)

	return <-ender.Ends()
}

type wsClientCommand struct {
	url     string
	spammer spammer
}

func (cl *wsClientCommand) Help() cmdy.Help { return cmdy.Synopsis("wsclient") }

func (cl *wsClientCommand) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	cl.spammer.Flags(flags)
	args.StringOptional(&cl.url, "url", "ws://localhost:9632/", "host")
}

func (cl *wsClientCommand) Run(ctx cmdy.Context) error {
	// defer profile.Start().Stop()

	dialer := cl.spammer.Dialer(negotiatorBuild())

	clientCb := func(handler sockerball.Handler, opts ...sockerball.ClientOption) (sockerball.Client, error) {
		wsDialer := websocket.Dialer{
			HandshakeTimeout: 5 * time.Second,
		}
		sock, _, err := wsDialer.Dial(cl.url, nil)
		if err != nil {
			return nil, err
		}
		client, err := dialer.Client(ctx, wsocketsrv.NewCommunicator(sock), handler, opts...)
		if err != nil {
			return nil, err
		}
		return client, nil
	}
	return cl.spammer.Spam(ctx, nil, clientCb)
}

type wsServerCommand struct {
	host string
}

func (sc *wsServerCommand) Help() cmdy.Help { return cmdy.Synopsis("wsserver") }

func (sc *wsServerCommand) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	flags.StringVar(&sc.host, "host", ":9632", "host")
}

func (sc *wsServerCommand) Run(ctx cmdy.Context) error {
	defer profile.Start().Stop()

	ln := wsocketsrv.NewListener(websocket.Upgrader{})

	websrv := &http.Server{
		Addr:    sc.host,
		Handler: ln,
	}

	ender := service.NewEndListener(1)
	websvc := service.New("", serviceutil.NewHTTP(websrv))
	if err := service.StartTimeout(5*time.Second, services.Runner(), websvc); err != nil {
		return err
	}

	handler := &ServerHandler{}

	srv := sockerball.NewServer(nil, ln, negotiatorBuild(), handler,
		sockerball.ServerConnect(func(srv *sockerball.Server, id sockerball.ConnID) {
			fmt.Println("connect", id)
		}),
		sockerball.ServerDisconnect(func(srv *sockerball.Server, id sockerball.ConnID, err error) {
			fmt.Println("disconnect", id, err)
		}),
	)
	svc := service.New(service.Name(sc.host), srv).WithEndListener(ender)
	if err := service.StartTimeout(5*time.Second, services.Runner(), svc); err != nil {
		return err
	}
	fmt.Printf("listening on %s\n", sc.host)

	return <-ender.Ends()
}

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

type JSONEnvelope struct {
	ID      sockerball.MessageID
	ReplyTo sockerball.MessageID
	Kind    int
	Message json.RawMessage
}

type ServerHandler struct{}

func (h ServerHandler) HandleRequest(ctx context.Context, in sockerball.IncomingRequest) (rs sockerball.Message, rerr error) {
	switch msg := in.Message.(type) {
	case *TestRequest:
		return &TestResponse{Bar: msg.Foo + ": yep!"}, nil
	default:
		return &OK{}, nil
	}
}

type OK struct{}

type TestRequest struct {
	Foo string
}

type TestResponse struct {
	Bar string
}

type TestCommand struct{}

type Mapper struct{}

func (m Mapper) Message(kind int) (sockerball.Message, error) {
	switch kind {
	case 1:
		return &TestRequest{}, nil
	case 2:
		return &TestResponse{}, nil
	case 3:
		return &TestCommand{}, nil
	case 4:
		return &OK{}, nil
	default:
		return nil, fmt.Errorf("unknown kind %d", kind)
	}
}

func (m Mapper) MessageKind(msg sockerball.Message) (int, error) {
	switch msg.(type) {
	case *TestRequest:
		return 1, nil
	case *TestResponse:
		return 2, nil
	case *TestCommand:
		return 3, nil
	case *OK:
		return 4, nil
	default:
		return 0, fmt.Errorf("unknown msg %T", msg)
	}
}

type Proto struct {
	version      int
	name         string
	mapper       sockerball.Mapper
	codec        sockerball.Codec
	messageLimit int
}

func (p Proto) ProtocolName() string      { return p.name }
func (p Proto) Mapper() sockerball.Mapper { return p.mapper }
func (p Proto) Codec() sockerball.Codec   { return p.codec }
func (p Proto) Version() int              { return p.version }
func (p Proto) MessageLimit() int         { return p.messageLimit }

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

func debugServer(host string) {
	mux := http.NewServeMux()
	mux.Handle("/debug/vars", expvar.Handler())
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	hsrv := &http.Server{Addr: host}
	hsrv.Handler = mux
	hsvc := &serviceutil.HTTP{Server: hsrv}
	svc := service.New("", hsvc)
	if err := service.StartTimeout(10*time.Second, services.Runner(), svc); err != nil {
		panic(err)
	}
	fmt.Println("debug server running on", hsvc.Port())
}
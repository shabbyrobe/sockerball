package example

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
)

func DebugServer(host string) {
	mux := http.NewServeMux()
	mux.Handle("/debug/vars", expvar.Handler())
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	hsrv := &http.Server{Addr: host}
	hsrv.Handler = mux
	go func() {
		if err := hsrv.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
	fmt.Println("debug server running on", host)
}

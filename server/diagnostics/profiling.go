package diagnostics

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"
)

// Profiling is opt-in and served by a separate loopback-only listener. It is
// never installed on the public/player HTTP mux or enabled by support reports.
func StartProfiling(address string) (func(), error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return nil, fmt.Errorf("profiling requires a numeric loopback address and port")
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return nil, fmt.Errorf("invalid profiling port")
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if value := r.URL.Query().Get("seconds"); value != "" {
			seconds, err := strconv.Atoi(value)
			if err != nil || seconds < 1 || seconds > 30 {
				http.Error(w, "seconds must be between 1 and 30", http.StatusBadRequest)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-store")
		mux.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 40 * time.Second, IdleTimeout: 10 * time.Second, MaxHeaderBytes: 8192}
	go server.Serve(listener)
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		server.Shutdown(ctx)
		server.Close()
	}, nil
}

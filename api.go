package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof" // register handlers
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func api(ctx context.Context, listen string, mux *http.ServeMux, metrics []prometheus.Collector) error {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorMemStatsMetricsDisabled(),
		collectors.WithGoCollectorRuntimeMetrics(
			collectors.GoRuntimeMetricsRule{
				Matcher: regexp.MustCompile(`^(/gc/gogc:percent|/gc/gomemlimit:bytes|/gc/heap/allocs:bytes|/gc/heap/allocs:objects|/gc/heap/goal:bytes|/memory/classes/heap/released:bytes|/memory/classes/heap/stacks:bytes|/memory/classes/total:bytes|/sched/gomaxprocs:threads|/sched/goroutines:goroutines|/sched/latencies:seconds)$`),
			},
		),
	))
	reg.MustRegister(metrics...)
	opts := promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, opts))
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	l, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("couldn't start API server: %w", err)
	}
	srv := http.Server{
		Handler:     mux,
		ReadTimeout: 5 * time.Second,
		BaseContext: func(l net.Listener) context.Context { return ctx },
	}
	go func() {
		slog.InfoContext(ctx, "HTTP API server", slog.Any("addr", l.Addr()))
		err := srv.Serve(l)
		if err == http.ErrServerClosed {
			return
		}
		slog.ErrorContext(ctx, "HTTP API server closed", slog.Any("err", err))
	}()
	<-ctx.Done()
	// The context is now done, so it is obviously the wrong choice for
	// managing the shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

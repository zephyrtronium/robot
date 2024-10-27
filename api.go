package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Userspace metrics.
var (
	tmiMsgsCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "robot",
		Subsystem: "tmi",
		Name:      "messages",
		Help:      "Number of PRIVMSGs received from TMI during the program lifetime.",
	})
	tmiCommandsCount = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "robot",
		Subsystem: "tmi",
		Name:      "commands",
		Help:      "Number of command invocations received in Twitch chat during the program lifetime.",
	})
)

func api(ctx context.Context, listen string, mux *http.ServeMux) error {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorMemStatsMetricsDisabled(),
		collectors.WithGoCollectorRuntimeMetrics(
			collectors.GoRuntimeMetricsRule{
				Matcher: regexp.MustCompile(`^(/gc/gogc:percent|/gc/gomemlimit:bytes|/gc/heap/allocs:bytes|/gc/heap/allocs:objects|/gc/heap/goal:bytes|/memory/classes/heap/released:bytes|/memory/classes/heap/stacks:bytes|/memory/classes/total:bytes|/sched/gomaxprocs:threads|/sched/goroutines:goroutines|/sched/latencies:seconds)$`),
			},
		),
	))
	reg.MustRegister(tmiMsgsCount)
	reg.MustRegister(tmiCommandsCount)
	opts := promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, opts))
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
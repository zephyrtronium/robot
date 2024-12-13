package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof" // register handlers
	"regexp"
	"strconv"
	"time"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/userhash"
)

func (robo *Robot) api(ctx context.Context, listen string, mux *http.ServeMux, metrics []prometheus.Collector) error {
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
	mux.HandleFunc("GET /api/message/{tag...}", robo.apiRecall)
	mux.HandleFunc("POST /api/message/{tag...}", robo.apiLearn)
	mux.HandleFunc("DELETE /api/message/{tag...}", robo.apiForget)
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

func jsonerror(w http.ResponseWriter, status int, msg string) {
	v := struct {
		Error  string `json:"error"`
		Status int    `json:"status"`
	}{
		Error:  msg,
		Status: status,
	}
	b, err := json.Marshal(&v)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(status)
	w.Write(b)
}

type apiMessage struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Time string `json:"time,omitzero"`
}

func (robo *Robot) apiRecall(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := slog.With(slog.String("api", "recall"), slog.Any("trace", uuid.New()))
	log.InfoContext(ctx, "handle", slog.String("route", r.Pattern), slog.String("remote", r.RemoteAddr))
	defer log.InfoContext(ctx, "done")
	w.Header().Set("Content-Type", "application/json")
	tag := r.PathValue("tag")
	page := r.FormValue("page")
	n := 64
	if s := r.FormValue("n"); s != "" {
		var err error
		n, err = strconv.Atoi(s)
		if err != nil || n <= 0 {
			log.WarnContext(ctx, "bad request", slog.String("n", s), slog.Any("err", err))
			jsonerror(w, http.StatusBadRequest, "invalid page size")
			return
		}
	}
	p := make([]brain.Message, n)
	log.InfoContext(ctx, "recall", slog.String("tag", tag), slog.String("page", page), slog.Int("n", n))
	n, next, err := robo.brain.Recall(ctx, tag, page, p)
	if err != nil {
		log.ErrorContext(ctx, "couldn't recall", slog.Any("err", err))
		jsonerror(w, http.StatusInternalServerError, err.Error())
		return
	}
	if n == 0 && page == "" {
		// We tried to start recollection, but there were no messages.
		// The tag must not exist.
		log.WarnContext(ctx, "no recollection")
		jsonerror(w, http.StatusNotFound, "no messages for tag")
		return
	}
	u := struct {
		Data   []apiMessage `json:"data"`
		Page   string       `json:"page,omitzero"`
		Status int          `json:"status"`
	}{
		Data:   make([]apiMessage, n),
		Page:   next,
		Status: http.StatusOK,
	}
	for i, m := range p[:n] {
		u.Data[i] = apiMessage{ID: m.ID, Text: m.Text}
		if m.Timestamp != 0 {
			u.Data[i].Time = m.Time().Format(time.RFC3339)
		}
	}
	b, err := json.Marshal(&u)
	if err != nil {
		panic(err)
	}
	if _, err := w.Write(b); err != nil {
		log.ErrorContext(ctx, "write response failed", slog.Any("err", err))
	}
}

func (robo *Robot) apiLearn(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := slog.With(slog.String("api", "recall"), slog.Any("trace", uuid.New()))
	log.InfoContext(ctx, "handle", slog.String("route", r.Pattern), slog.String("remote", r.RemoteAddr))
	defer log.InfoContext(ctx, "done")
	tag := r.PathValue("tag")
	d := jsontext.NewDecoder(r.Body)
	var all error
	var msg apiMessage
	for {
		err := json.UnmarshalDecode(d, &msg)
		switch err {
		case nil: // do nothing
		case io.EOF:
			// Done; transmit any learn errors.
			if all != nil {
				jsonerror(w, http.StatusInternalServerError, all.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		default:
			log.ErrorContext(ctx, "read message", slog.Any("err", err))
			jsonerror(w, http.StatusBadRequest, "message read failed")
			return
		}
		m := brain.Message{
			ID:     msg.ID,
			Sender: userhash.Hash{'A', 'P', 'I'},
			Text:   msg.Text,
		}
		if m.ID == "" {
			m.ID = "API:" + uuid.NewString()
		}
		if msg.Time == "" {
			m.Timestamp = time.Now().UnixMilli()
		} else {
			t, err := time.Parse(time.RFC3339, msg.Time)
			if err != nil {
				all = errors.Join(all, err)
				continue
			}
			m.Timestamp = t.UnixMilli()
		}
		if err := brain.Learn(ctx, robo.brain, tag, &m); err != nil {
			log.ErrorContext(ctx, "learn failed", slog.String("tag", tag), slog.String("id", m.ID), slog.Any("err", err))
			all = errors.Join(all, err)
			// continue on
		}
	}
}

func (robo *Robot) apiForget(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := slog.With(slog.String("api", "recall"), slog.Any("trace", uuid.New()))
	log.InfoContext(ctx, "handle", slog.String("route", r.Pattern), slog.String("remote", r.RemoteAddr))
	defer log.InfoContext(ctx, "done")
	tag := r.PathValue("tag")
	d := jsontext.NewDecoder(r.Body)
	var all error
	for {
		tok, err := d.ReadToken()
		switch err {
		case nil: // do nothing
		case io.EOF:
			// Done; transmit any forget errors.
			if all != nil {
				jsonerror(w, http.StatusInternalServerError, all.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		default:
			log.ErrorContext(ctx, "read token", slog.Any("err", err))
			jsonerror(w, http.StatusBadRequest, "token read failed")
			return
		}
		if tok.Kind() != '"' {
			log.WarnContext(ctx, "invalid token", slog.Any("kind", tok.Kind()))
			jsonerror(w, http.StatusBadRequest, "input not a JSON string")
			return
		}
		id := tok.String()
		if err := robo.brain.Forget(ctx, tag, id); err != nil {
			log.ErrorContext(ctx, "forget failed", slog.String("tag", tag), slog.String("id", id), slog.Any("err", err))
			all = errors.Join(all, err)
			// continue on
		}
	}
}

package dash

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/zephyrtronium/robot/brain"
)

type ownerHandler struct {
	br *brain.Brain
}

type ownerFormatter struct {
	Me       string
	Messages []messageItem
}

type messageItem struct {
	Time string
	Ch   string
	Text string
	Tags string
}

const ownerTemplSrc = `<!DOCTYPE html>
<title>{{.Me}}</title>
<link rel="preconnect" href="https://fonts.gstatic.com">
<link href="https://fonts.googleapis.com/css2?family=Source+Sans+Pro:ital@0;1&display=swap" rel="stylesheet"> 
<style>
html {
	font-family: 'Source Sans Pro', sans-serif;
	background-color: #0e0e10;
	color: #efeff1;
}

.messages {
	display: flex;
	flex-direction: column-reverse;
	margin: 0 auto;
	max-width: 75%;
	max-height: 98vh;
	overflow-y: scroll;
}

.message {
	display: flex;
	flex: none;
	align-items: center;
	color: #afafb1;
}

.message > div {
	padding: 0px 8px;
}

.message > div > p {
	margin: 0px;
}

.message__time {
	flex-basis: 4em;
	text-align: right;
}

.message__text {
	color: #efeff1;
	flex: auto;
	flex-basis: fill;
}

.message:hover {
	padding: 1em 0px;
}

.message > .message__tags {
	display: none;
}

.message:hover > .message__tags {
	display: inline;
	max-width: 50%;
	overflow-wrap: anywhere;
	font-style: italic;
	font-size: small;
	text-align: right;
}
</style>
<div class="messages">
{{range .Messages}}	<div class="message">
		<div class="message__time"><p>{{.Time}}</div>
		<div class="message__ch"><p>{{.Ch}}</div>
		<div class="message__text"><p>{{.Text}}</div>
		<div class="message__tags"><p>{{.Tags}}</div>
	</div>
{{end}}</div>
`

var ownerTempl = template.Must(template.New("owner").Parse(ownerTemplSrc))

func (h *ownerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *ownerHandler) get(w http.ResponseWriter) {
	failed := func(err error, where string) {
		if err == nil {
			panic(fmt.Errorf("nil error in failed from %q", where))
		}
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "<html><body><p>%s failed: %v</p></body></html>", where, template.HTMLEscapeString(err.Error()))
	}
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("X-Content-Type-Options", "nosniff")
	rows, err := h.br.Query(context.TODO(), `SELECT tags, time, chan, msg FROM history ORDER BY time DESC`)
	if err != nil {
		failed(err, "history query")
		return
	}
	f := ownerFormatter{Me: h.br.Name()}
	for rows.Next() {
		var tags, ch, msg string
		var tm time.Time
		if err := rows.Scan(&tags, &tm, &ch, &msg); err != nil {
			failed(err, "history row scan")
			return
		}
		f.Messages = append(f.Messages, messageItem{Time: tm.Format("15:04:05"), Ch: ch, Text: msg, Tags: tags})
	}
	if rows.Err() != nil {
		failed(rows.Err(), "history read")
		return
	}
	if err := ownerTempl.Execute(w, f); err != nil {
		failed(err, "writing body")
	}
}

// Owner starts the owner dashboard.
func Owner(ctx context.Context, wg *sync.WaitGroup, br *brain.Brain, addr string, tls *tls.Config, log *log.Logger) error {
	srv := http.Server{
		Addr:              addr,
		Handler:           &ownerHandler{br: br},
		TLSConfig:         tls,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ErrorLog:          log,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	go func() {
		<-ctx.Done()
		srv.Shutdown(ctx)
		wg.Done()
	}()
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

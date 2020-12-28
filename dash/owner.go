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

const ownerTemplSrc = `
<http>
<head>
<title>{{.Me}}</title>
<style>
	.messages {
		display: flex;
		flex-direction: column;
	}
	
	.message {
		border: 1px solid black;
		display: flex;
		flex: none;
	}
	
	.message > div {
		flex: none;
		padding: 0px 8px;
	}
	
	.message > .message__tags {
		display: none;
	}
	
	.message:hover > .message__tags {
		display: contents;
	}
</style>
</head>
<body>
<div class="messages">
{{range .Messages}}	<div class="message">
		<div class="message__time"><p>{{.Time}}</p></div>
		<div class="message__ch"><p>{{.Ch}}</p></div>
		<div class="message__text"><p>{{.Text}}</p></div>
		<div class="message__tags"><p>{{.Tags}}</p></div>
	</div>
{{end}}</div>
</body>
</http>
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
	rows, err := h.br.Query(context.TODO(), `SELECT tags, time, chan, msg FROM history`)
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

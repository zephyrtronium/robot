package twitch

import (
	"context"
	"embed"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

type reqspy struct {
	// got is the first request the round tripper received.
	got *http.Request
	// respond is the response the round tripper returns.
	respond *http.Response
}

func (r *reqspy) RoundTrip(req *http.Request) (*http.Response, error) {
	if r.got != nil {
		return nil, errors.New("already have a request")
	}
	r.got = req
	return r.respond, nil
}

//go:embed testdata/*.json
var jsonFiles embed.FS

// apiresp creates a reqspy responding with the given testdata document.
func apiresp(status int, file string) *reqspy {
	f, err := jsonFiles.Open(path.Join("testdata/", file))
	if err != nil {
		panic(err)
	}
	return &reqspy{
		respond: &http.Response{
			StatusCode: status,
			Body:       f,
		},
	}
}

func TestReqJSON(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		spy := &reqspy{
			respond: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"data":1}`)),
			},
		}
		cl := Client{
			HTTP: &http.Client{
				Transport: spy,
			},
			ID: "bocchi",
		}
		tok := &oauth2.Token{AccessToken: "ryo"}
		var u int
		pag, err := reqjson(context.Background(), cl, tok, "GET", "https://bocchi.rocks/bocchi", &u)
		if err != nil {
			t.Errorf("failed to request: %v", err)
		}
		if u != 1 {
			t.Errorf("didn't get the result: want 1, got %d", u)
		}
		if pag != "" {
			t.Errorf("unexpected pagination cursor %q", pag)
		}
		if got := spy.got.URL.String(); got != "https://bocchi.rocks/bocchi" {
			t.Errorf(`request went to the wrong place: want "https://bocchi.rocks/bocchi", got %q`, got)
		}
		if got := spy.got.Header.Get("Authorization"); got != "Bearer ryo" {
			t.Errorf(`wrong authorization: want "Bearer ryo", got %q`, got)
		}
		if got := spy.got.Header.Get("Client-Id"); got != "bocchi" {
			t.Errorf(`wrong client-id: want "bocchi", got %q`, got)
		}
	})
	t.Run("expired", func(t *testing.T) {
		spy := &reqspy{
			respond: &http.Response{
				StatusCode: 401,
				Body:       io.NopCloser(strings.NewReader(`{"data":1}`)),
			},
		}
		cl := Client{
			HTTP: &http.Client{
				Transport: spy,
			},
			ID: "bocchi",
		}
		tok := &oauth2.Token{AccessToken: "ryo"}
		var u int
		_, err := reqjson(context.Background(), cl, tok, "GET", "https://bocchi.rocks/bocchi", &u)
		if !errors.Is(err, ErrNeedRefresh) {
			t.Errorf("unauthorized request didn't return ErrNeedRefresh error")
		}
	})
	t.Run("paginated", func(t *testing.T) {
		spy := &reqspy{
			respond: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"data":1,"pagination":{"cursor":"bocchi"}}`)),
			},
		}
		cl := Client{
			HTTP: &http.Client{Transport: spy},
			ID:   "bocchi",
		}
		tok := &oauth2.Token{AccessToken: "ryo"}
		var u int
		pag, err := reqjson(context.Background(), cl, tok, "GET", "https://bocchi.rocks/bocchi", &u)
		if err != nil {
			t.Errorf("failed to request: %v", err)
		}
		if pag != "bocchi" {
			t.Errorf("wrong pagination cursor: want %q, got %q", "bocchi", pag)
		}
	})
}

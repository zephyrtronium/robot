package twitch

import (
	"context"
	_ "embed"
	"errors"
	"io"
	"net/http"
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
			Token: &oauth2.Token{
				AccessToken: "bocchi",
			},
		}
		var u int
		err := reqjson(context.Background(), cl, "GET", "https://bocchi.rocks/bocchi", nil, &u)
		if err != nil {
			t.Errorf("failed to request: %v", err)
		}
		if u != 1 {
			t.Errorf("didn't get the result: want 1, got %d", u)
		}
		if got := spy.got.URL.String(); got != "https://bocchi.rocks/bocchi" {
			t.Errorf(`request went to the wrong place: want "https://bocchi.rocks/bocchi", got %q`, got)
		}
		if got := spy.got.Header.Get("Authorization"); got != "Bearer bocchi" {
			t.Errorf(`wrong authorization: want "Bearer bocchi", got %q`, got)
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
			Token: &oauth2.Token{
				AccessToken: "bocchi",
			},
		}
		var u int
		err := reqjson(context.Background(), cl, "GET", "https://bocchi.rocks/bocchi", nil, &u)
		if !errors.Is(err, ErrNeedRefresh) {
			t.Errorf("unauthorized request didn't return ErrNeedRefresh error")
		}
	})
}

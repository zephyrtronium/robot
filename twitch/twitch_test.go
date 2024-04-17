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
	if spy.got.URL.String() != "https://bocchi.rocks/bocchi" {
		t.Errorf("request went to the wrong place: want https://bocchi.rocks/bocchi, got %v", spy.got.URL)
	}
}

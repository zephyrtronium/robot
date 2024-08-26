package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func hostCCF(t *testing.T) oauth2.Config {
	t.Helper()
	var mux http.ServeMux
	tokenResp := &fixedResponse{
		status: 200,
		body:   `{"access_token":"nijika","expires_in":5011271,"token_type":"bearer"}`,
	}
	mux.Handle("POST /oauth2/token", tokenResp)
	srv := httptest.NewServer(&mux)
	t.Cleanup(srv.Close)
	tokenURL, err := url.JoinPath(srv.URL, "/oauth2/token")
	if err != nil {
		panic(err)
	}
	cfg := oauth2.Config{
		ClientID:     "bocchi",
		ClientSecret: "ryo",
		Endpoint: oauth2.Endpoint{
			TokenURL: tokenURL,
		},
		Scopes: []string{"chat:read", "chat:edit"},
	}
	return cfg
}

func TestCCF(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cfg := hostCCF(t)
	src := ClientCredentialsFlow(cfg, &http.Client{Timeout: 30 * time.Second}).(*ccf)
	tok, err := src.flowLocked(ctx)
	if err != nil {
		t.Errorf("couldn't get token: %v", err)
	}
	if tok.AccessToken != "nijika" {
		t.Errorf("wrong access token: want %q, got %q", "nijika", tok.AccessToken)
	}
}

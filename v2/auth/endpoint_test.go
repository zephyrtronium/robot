package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
)

// testEndpoint is a mini OAuth2 authorization code grant server for tests.
type testEndpoint struct {
	// code is the authorization code passed to the callback, as well as to
	// check against in the token endpoint.
	code string
	// client and secret are the client ID and client secret to expect.
	client, secret string
	// access and refresh are the OAuth2 tokens to provide.
	access, refresh string
	// authorizes and tokens are call counters for authorize and token.
	authorizes, tokens atomic.Uint32
}

func (e *testEndpoint) server() (*httptest.Server, oauth2.Endpoint) {
	r := chi.NewRouter()
	r.Get("/oauth2/authorize", e.authorize)
	r.Post("/oauth2/token", e.token)
	s := httptest.NewServer(r)
	ep := oauth2.Endpoint{
		AuthURL:  s.URL + "/oauth2/authorize",
		TokenURL: s.URL + "/oauth2/token",
	}
	return s, ep
}

func (e *testEndpoint) authorize(w http.ResponseWriter, r *http.Request) {
	e.authorizes.Add(1)
	cb, err := url.Parse(r.FormValue("redirect_uri"))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	v := url.Values{
		"state": {r.FormValue("state")},
	}
	if t := r.FormValue("response_type"); t == "code" {
		v.Set("code", e.code)
		v.Set("scope", r.FormValue("scope"))
	} else {
		v.Set("error", "access_denied")
		v.Set("error_description", "invalid response_type")
	}
	cb.RawQuery = v.Encode()
	http.Redirect(w, r, cb.String(), http.StatusTemporaryRedirect)
}

func (e *testEndpoint) token(w http.ResponseWriter, r *http.Request) {
	e.tokens.Add(1)
	w.Header().Set("Content-Type", "application/json")
	type wrong struct {
		Error   string `json:"error"`
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	fail := func(w http.ResponseWriter, e, msg string) {
		resp := wrong{
			Error:   e,
			Status:  http.StatusBadRequest,
			Message: msg,
		}
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			panic(err)
		}
	}
	if r.Method != "POST" {
		fail(w, "wrong method", fmt.Sprintf("want POST, got %s", r.Method))
		return
	}
	client, secret := r.FormValue("client_id"), r.FormValue("client_secret")
	if client != e.client || secret != e.secret {
		s := fmt.Sprintf("want %s %s\ngot  %s %s", e.client, e.secret, client, secret)
		fail(w, "wrong client/secret", s)
		return
	}

	type resp struct {
		Access    string   `json:"access_token"`
		Refresh   string   `json:"refresh_token"`
		ExpiresIn int      `json:"expires_in,omitempty"`
		Scope     []string `json:"scope"`
		Type      string   `json:"token_type"`
	}
	grant := r.FormValue("grant_type")
	switch grant {
	case "authorization_code":
		resp := resp{
			Access:    e.access,
			Refresh:   e.refresh,
			ExpiresIn: 1234,
			Scope:     []string{"chat:read", "chat:edit"},
			Type:      "bearer",
		}
		json.NewEncoder(w).Encode(resp)
	case "refresh_token":
		resp := resp{
			Access:  e.access,
			Refresh: e.refresh,
			Scope:   []string{"chat:read", "chat:edit"},
			Type:    "bearer",
		}
		json.NewEncoder(w).Encode(resp)
	default:
		fail(w, "bad grant_type", grant)
	}
}

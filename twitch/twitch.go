package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
)

// Client holds the context for requests to the Twitch API.
type Client struct {
	// HTTP is the HTTP client for performing requests.
	// If nil, http.DefaultClient is used.
	HTTP *http.Client
	// ID is the application's client ID.
	ID string
}

// reqjson performs an HTTP request and decodes the response as JSON.
// The response body is truncated to 2 MB.
func reqjson[Resp any](ctx context.Context, client Client, tok *oauth2.Token, method, url string, body io.Reader, u *Resp) error {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("couldn't make request: %w", err)
	}
	tok.SetAuthHeader(req)
	req.Header.Set("Client-Id", client.ID)
	hc := client.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't %s: %w", method, err)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return fmt.Errorf("couldn't read response: %w", err)
	}
	resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK: // do nothing
	case http.StatusUnauthorized:
		return fmt.Errorf("request failed: %s (%w)", b, ErrNeedRefresh)
	default:
		return fmt.Errorf("request failed: %s (%s)", b, resp.Status)
	}
	r := struct {
		Data *Resp `json:"data"`
	}{u}
	if err := json.Unmarshal(b, &r); err != nil {
		return fmt.Errorf("couldn't decode JSON response: %w", err)
	}
	return nil
}

// apiurl creates an api.twitch.tv URL for the given endpoint and with the
// given URL parameters.
func apiurl(ep string, values url.Values) string {
	u, err := url.JoinPath("https://api.twitch.tv/", ep)
	if err != nil {
		panic("twitch: bad url join with " + ep)
	}
	if len(values) == 0 {
		return u
	}
	return u + "?" + values.Encode()
}

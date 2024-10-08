package twitch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
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
// The returned string is the pagination cursor, if any.
// The response body is truncated to 2 MB.
func reqjson[Resp any](ctx context.Context, client Client, tok *oauth2.Token, method, url string, u *Resp) (jsontext.Value, error) {
	return reqjsonbody(ctx, client, tok, method, url, "", nil, u)
}

// reqjsonbody is like reqjson but takes a request body and content type.
func reqjsonbody[Resp any](ctx context.Context, client Client, tok *oauth2.Token, method, url, content string, body io.Reader, u *Resp) (jsontext.Value, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("couldn't make request: %w", err)
	}
	tok.SetAuthHeader(req)
	req.Header.Set("Client-Id", client.ID)
	if content != "" {
		req.Header.Set("Content-Type", content)
	}
	hc := client.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't %s: %w", method, err)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("couldn't read response: %w", err)
	}
	resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted: // do nothing
	case http.StatusNoContent:
		// We could verify that the response type is supposed to be empty,
		// but reflecting would be a lot of work for a response that indicates
		// no work, not to mention it would be fragile in consideration of
		// e.g. unexported fields.
		// Instead, just zero the response.
		*u = *new(Resp)
		return nil, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("request failed: %s (%w)", b, ErrNeedRefresh)
	default:
		return nil, fmt.Errorf("request failed: %s (%s)", b, resp.Status)
	}
	r := struct {
		Data *Resp          `json:"data"`
		Rest jsontext.Value `json:",unknown"`
	}{Data: u}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("couldn't decode JSON response: %w", err)
	}
	return r.Rest, nil
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

func pagination(rest jsontext.Value) (string, error) {
	var u struct {
		Pagination struct {
			Cursor string `json:"cursor"`
		} `json:"pagination"`
	}
	err := json.Unmarshal([]byte(rest), &u)
	return u.Pagination.Cursor, err
}

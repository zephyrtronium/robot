package twitch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"strings"

	"github.com/go-json-experiment/json"
	"golang.org/x/oauth2"
)

// Conduit is the EventSub API representation of a conduit.
type Conduit struct {
	ID         string `json:"id"`
	ShardCount int    `json:"shard_count"`
}

// Conduits calls the Get Conduits API to list conduits owned by the client.
// Requires an app access token.
func Conduits(ctx context.Context, client Client, tok *oauth2.Token) ([]Conduit, error) {
	url := apiurl("/helix/eventsub/conduits", nil)
	resp := make([]Conduit, 0, 5) // api limits to 5 conduits
	_, err := reqjson(ctx, client, tok, "GET", url, &resp)
	if err != nil {
		return nil, fmt.Errorf("couldn't get conduits: %w", err)
	}
	return resp, nil
}

// CreateConduit calls the Create Conduits API create a new conduit.
// Requires an app access token.
func CreateConduit(ctx context.Context, client Client, tok *oauth2.Token, shards int) (Conduit, error) {
	body := fmt.Sprintf(`{"shard_count":%d}`, shards) // json the easy way
	url := apiurl("/helix/eventsub/conduits", nil)
	var resp []Conduit
	_, err := reqjsonbody(ctx, client, tok, "POST", url, "application/json", strings.NewReader(body), &resp)
	if err != nil {
		return Conduit{}, fmt.Errorf("couldn't create conduit: %w", err)
	}
	if len(resp) != 1 {
		return Conduit{}, fmt.Errorf("somehow created %d conduits", len(resp))
	}
	return resp[0], nil
}

// UpdateConduit calls the Update Conduits API to resize a conduit.
// Requires an app access token.
func UpdateConduit(ctx context.Context, client Client, tok *oauth2.Token, conduit string, shards int) (Conduit, error) {
	body, err := json.Marshal(Conduit{ID: conduit, ShardCount: shards})
	if err != nil {
		// should never happen
		panic(err)
	}
	url := apiurl("/helix/eventsub/conduits", nil)
	var resp []Conduit
	_, err = reqjsonbody(ctx, client, tok, "PATCH", url, "application/json", bytes.NewReader(body), &resp)
	if err != nil {
		return Conduit{}, fmt.Errorf("couldn't update conduit: %w", err)
	}
	if len(resp) != 1 {
		return Conduit{}, fmt.Errorf("somehow updated %d conduits", len(resp))
	}
	return resp[0], nil
}

// Shard is the EventSub API representation of a conduit shard.
type Shard struct {
	ID           int             `json:"id,string"`
	Status       string          `json:"status,omitempty"`
	Transport    *ShardTransport `json:"transport,omitempty"`
	Disconnected string          `json:"disconnected_at,omitempty"`
}

type ShardTransport struct {
	Method       string `json:"method"`
	Callback     string `json:"callback,omitempty"`
	Session      string `json:"session_id,omitempty"`
	Connected    string `json:"connected_at,omitempty"`
	Disconnected string `json:"disconnected_at,omitempty"`
}

// Shards calls the Get Conduit Shards API to list a conduit's shards.
// Results are yielded in groups as they are obtained from the API.
// The slice passed to the yield function must not be retained.
// Requires an app access token.
func Shards(ctx context.Context, client Client, tok *oauth2.Token, conduit, status string) iter.Seq2[[]Shard, error] {
	return func(yield func([]Shard, error) bool) {
		vals := url.Values{"conduit_id": {"conduit"}}
		if status != "" {
			vals["status"] = []string{status}
		}
		var resp []Shard
		for {
			url := apiurl("/helix/eventsub/conduits/shards", vals)
			rest, err := reqjson(ctx, client, tok, "GET", url, &resp)
			if err != nil {
				yield(nil, fmt.Errorf("couldn't get shards: %w", err))
				return
			}
			pag, err := pagination(rest)
			if err != nil {
				yield(nil, fmt.Errorf("couldn't get pagination: %w", err))
				return
			}
			if pag == "" {
				return
			}
			vals["after"] = []string{pag}
		}
	}
}

// ShardUpdate describes a change to a shard's transport.
type ShardUpdate struct {
	ID       int
	Method   string
	Callback string
	Secret   string
	Session  string
}

// UpdateShards calls the Update Conduit Shards API to modify transports of
// conduit shards.
// Requires an app access token.
func UpdateShards(ctx context.Context, client Client, tok *oauth2.Token, conduit string, updates []ShardUpdate) ([]Shard, error) {
	type transport struct {
		Method   string `json:"method"`
		Callback string `json:"callback,omitempty"`
		Secret   string `json:"secret,omitempty"`
		Session  string `json:"session_id,omitempty"`
	}
	type shards struct {
		ID        int       `json:"id,string"`
		Transport transport `json:"transport"`
	}
	d := struct {
		Conduit string   `json:"conduit_id"`
		Shards  []shards `json:"shards"`
	}{
		Conduit: conduit,
		Shards:  make([]shards, len(updates)),
	}
	for i, u := range updates {
		d.Shards[i] = shards{
			ID: u.ID,
			Transport: transport{
				Method:   u.Method,
				Callback: u.Callback,
				Secret:   u.Secret,
				Session:  u.Session,
			},
		}
	}
	body, err := json.Marshal(d)
	if err != nil {
		// should never happen
		panic(err)
	}
	url := apiurl("/helix/eventsub/conduits/shards", nil)
	var resp []Shard
	rest, err := reqjsonbody(ctx, client, tok, "PATCH", url, "application/json", bytes.NewReader(body), &resp)
	if err != nil {
		return nil, fmt.Errorf("couldn't update shards: %w", err)
	}
	if len(rest) != 0 {
		// Extra is error messages.
		var ee struct {
			Errors []struct {
				ID      int    `json:"id,string"`
				Message string `json:"message"`
				Code    string `json:"code"`
			} `json:"errors"`
		}
		if err := json.Unmarshal([]byte(rest), &ee); err != nil {
			return resp, fmt.Errorf("couldn't unmarshal errors from %q: %w", rest, err)
		}
		for _, e := range ee.Errors {
			err = errors.Join(err, fmt.Errorf("shard %d: %s (%s)", e.ID, e.Message, e.Code))
		}
	}
	return resp, err
}

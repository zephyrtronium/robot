package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	err := reqjson(ctx, client, tok, "GET", url, &resp)
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
	err := reqjsonbody(ctx, client, tok, "POST", url, "application/json", strings.NewReader(body), &resp)
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
	err = reqjsonbody(ctx, client, tok, "PATCH", url, "application/json", bytes.NewReader(body), &resp)
	if err != nil {
		return Conduit{}, fmt.Errorf("couldn't update conduit: %w", err)
	}
	if len(resp) != 1 {
		return Conduit{}, fmt.Errorf("somehow updated %d conduits", len(resp))
	}
	return resp[0], nil
}

package twitch

import (
	"context"
	"fmt"
	"iter"
	"net/url"

	"github.com/go-json-experiment/json/jsontext"
	"golang.org/x/oauth2"
)

type Subscription struct {
	ID        string                `json:"id"`
	Status    string                `json:"status"`
	Type      string                `json:"type"`
	Version   string                `json:"version"`
	Condition SubscriptionCondition `json:"condition"`
	Created   string                `json:"created_at"`
	Transport SubscriptionTransport `json:"transport"`
	Cost      int                   `json:"cost"`
}

type SubscriptionCondition struct {
	// Broadcaster is the broadcaster user ID for the condition.
	Broadcaster string `json:"broadcaster_user_id,omitempty"`
	// User is the user ID for the condition.
	User string `json:"user_id,omitempty"`
	// Extra holds any additional fields in the condition.
	Extra jsontext.Value `json:",unknown"`
}

type SubscriptionTransport struct {
	Method       string `json:"method"`
	Callback     string `json:"callback,omitempty"`
	Session      string `json:"session,omitempty"`
	Connected    string `json:"connected_at,omitempty"`
	Disconnected string `json:"disconnected_at,omitempty"`
	Conduit      string `json:"conduit_id,omitempty"`
}

// Subscriptions yields all EventSub subscriptions associated with a client
// of a given type. If typ is the empty string, all subscriptions are yielded.
// Requires an app access token for webhook or conduit subscriptions
// or a user access token for WebSocket subscriptions.
func Subscriptions(ctx context.Context, cl Client, tok *oauth2.Token, typ string) iter.Seq2[Subscription, error] {
	return func(yield func(Subscription, error) bool) {
		vals := make(url.Values, 2)
		if typ != "" {
			vals["type"] = []string{typ}
		}
		var resp []Subscription
		for {
			url := apiurl("/helix/eventsub/subscriptions", vals)
			rest, err := reqjson(ctx, cl, tok, "GET", url, &resp)
			if err != nil {
				yield(Subscription{}, fmt.Errorf("couldn't get subscriptions: %w", err))
				return
			}
			pag, err := pagination(rest)
			if err != nil {
				yield(Subscription{}, fmt.Errorf("couldn't get pagination: %w", err))
				return
			}
			for _, s := range resp {
				if !yield(s, nil) {
					return
				}
			}
			if pag == "" {
				return
			}
			vals["after"] = []string{pag}
		}
	}
}

// DeleteSubscription calls the Delete EventSub Subscription API to delete a subscription.
// Requires an app access token for webhook or conduit subscriptions
// or a user access token for WebSocket subscriptions.
func DeleteSubscription(ctx context.Context, cl Client, tok *oauth2.Token, id string) error {
	vals := url.Values{
		"id": {id},
	}
	url := apiurl("/helix/eventsub/subscriptions", vals)
	_, err := reqjson(ctx, cl, tok, "DELETE", url, new(struct{}))
	if err != nil {
		return fmt.Errorf("couldn't delete EventSub subscription: %w", err)
	}
	return nil
}

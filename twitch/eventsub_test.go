package twitch

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2"
)

func TestSubscriptions(t *testing.T) {
	spy := apiresp(200, "get-eventsub-subscriptions.json")
	cl := Client{
		HTTP: &http.Client{Transport: spy},
	}
	tok := &oauth2.Token{AccessToken: "bocchi"}
	want := []Subscription{
		{
			ID:      "26b1c993-bfcf-44d9-b876-379dacafe75a",
			Status:  "enabled",
			Type:    "stream.online",
			Version: "1",
			Condition: SubscriptionCondition{
				Broadcaster: "1234",
			},
			Created: "2020-11-10T20:08:33.12345678Z",
			Transport: SubscriptionTransport{
				Method:   "webhook",
				Callback: "https://this-is-a-callback.com",
			},
			Cost: 1,
		},
		{
			ID:      "35016908-41ff-33ce-7879-61b8dfc2ee16",
			Status:  "webhook_callback_verification_pending",
			Type:    "user.update",
			Version: "1",
			Condition: SubscriptionCondition{
				User: "1234",
			},
			Created: "2020-11-10T14:32:18.730260295Z",
			Transport: SubscriptionTransport{
				Method:   "webhook",
				Callback: "https://this-is-a-callback.com",
			},
			Cost: 0,
		},
	}
	var got []Subscription
	for s, err := range Subscriptions(context.Background(), cl, tok, "") {
		if err != nil {
			t.Error(err)
		}
		got = append(got, s)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("wrong result (+got/-want):\n%s", diff)
	}
}

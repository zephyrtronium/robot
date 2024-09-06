package twitch

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2"
)

func TestConduits(t *testing.T) {
	spy := apiresp(200, "get-conduits.json")
	cl := Client{
		HTTP: &http.Client{Transport: spy},
	}
	tok := &oauth2.Token{AccessToken: "bocchi"}
	got, err := Conduits(context.Background(), cl, tok)
	if err != nil {
		t.Error(err)
	}
	want := []Conduit{
		{ID: "26b1c993-bfcf-44d9-b876-379dacafe75a", ShardCount: 15},
		{ID: "bfcfc993-26b1-b876-44d9-afe75a379dac", ShardCount: 5},
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("wrong results (+got/-want):\n%s", diff)
	}
	if spy.got.Method != "GET" {
		t.Errorf("request was %s, not GET", spy.got.Method)
	}
}

func TestCreateConduit(t *testing.T) {
	spy := apiresp(200, "modify-conduits.json")
	cl := Client{
		HTTP: &http.Client{Transport: spy},
	}
	tok := &oauth2.Token{AccessToken: "bocchi"}
	got, err := CreateConduit(context.Background(), cl, tok, 5)
	if err != nil {
		t.Error(err)
	}
	want := Conduit{
		ID:         "bfcfc993-26b1-b876-44d9-afe75a379dac",
		ShardCount: 5,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("wrong result (+got/-want):\n%s", diff)
	}
	if spy.got.Method != "POST" {
		t.Errorf("request was %s, not POST", spy.got.Method)
	}
	if got := spy.got.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("content type was %s, not application/json", got)
	}
}

func TestUpdateConduit(t *testing.T) {
	spy := apiresp(200, "modify-conduits.json")
	cl := Client{
		HTTP: &http.Client{Transport: spy},
	}
	tok := &oauth2.Token{AccessToken: "bocchi"}
	id := "bfcfc993-26b1-b876-44d9-afe75a379dac"
	got, err := UpdateConduit(context.Background(), cl, tok, id, 5)
	if err != nil {
		t.Error(err)
	}
	want := Conduit{
		ID:         "bfcfc993-26b1-b876-44d9-afe75a379dac",
		ShardCount: 5,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("wrong result (+got/-want):\n%s", diff)
	}
	if spy.got.Method != "PATCH" {
		t.Errorf("request was %s, not PATCH", spy.got.Method)
	}
	if got := spy.got.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("content type was %s, not application/json", got)
	}
}

func TestShards(t *testing.T) {
	spy := apiresp(200, "get-conduit-shards.json")
	cl := Client{
		HTTP: &http.Client{Transport: spy},
	}
	tok := &oauth2.Token{AccessToken: "bocchi"}
	id := "bfcfc993-26b1-b876-44d9-afe75a379dac"
	want := []Shard{
		{
			ID:     0,
			Status: "enabled",
			Transport: &ShardTransport{
				Method:   "webhook",
				Callback: "https://this-is-a-callback.com",
			},
		},
		{
			ID:     1,
			Status: "webhook_callback_verification_pending",
			Transport: &ShardTransport{
				Method:   "webhook",
				Callback: "https://this-is-a-callback-2.com",
			},
		},
		{
			ID:     2,
			Status: "enabled",
			Transport: &ShardTransport{
				Method:    "websocket",
				Session:   "9fd5164a-a958-4c60-b7f4-6a7202506ca0",
				Connected: "2020-11-10T14:32:18.730260295Z",
			},
		},
		{
			ID:     3,
			Status: "enabled",
			Transport: &ShardTransport{
				Method:    "websocket",
				Session:   "238b4b08-13f1-4b8f-8d31-56665a7a9d9f",
				Connected: "2020-11-10T14:32:18.730260295Z",
			},
		},
		{
			ID:     4,
			Status: "websocket_disconnected",
			Transport: &ShardTransport{
				Method:       "websocket",
				Session:      "ad1c9fc3-0d99-4eb7-8a04-8608e8ff9ec9",
				Connected:    "2020-11-10T14:32:18.730260295Z",
				Disconnected: "2020-11-11T14:32:18.730260295Z",
			},
		},
	}
	for got, err := range Shards(context.Background(), cl, tok, id, "") {
		if err != nil {
			t.Error(err)
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("wrong result (+got/-want):\n%s", diff)
		}
	}
}

func TestUpdateShards(t *testing.T) {
	spy := apiresp(200, "update-conduit-shards.json")
	cl := Client{
		HTTP: &http.Client{Transport: spy},
	}
	tok := &oauth2.Token{AccessToken: "bocchi"}
	id := "bfcfc993-26b1-b876-44d9-afe75a379dac"
	up := []ShardUpdate{
		{
			ID:       0,
			Method:   "webhook",
			Callback: "https://this-is-a-callback.com",
			Secret:   "s3cre7",
		},
		{
			ID:       1,
			Method:   "webhook",
			Callback: "https://this-is-a-callback-2.com",
			Secret:   "s3cre7",
		},
		{
			ID:       3,
			Method:   "webhook",
			Callback: "https://this-is-a-callback-3.com",
			Secret:   "s3cre7",
		},
	}
	want := []Shard{
		{
			ID:     0,
			Status: "enabled",
			Transport: &ShardTransport{
				Method:   "webhook",
				Callback: "https://this-is-a-callback.com",
			},
		},
		{
			ID:     1,
			Status: "webhook_callback_verification_pending",
			Transport: &ShardTransport{
				Method:   "webhook",
				Callback: "https://this-is-a-callback-2.com",
			},
		},
	}
	got, err := UpdateShards(context.Background(), cl, tok, id, up)
	if err != nil {
		msg := err.Error()
		if !strings.Contains(msg, "3") || !strings.Contains(msg, "invalid_parameter") {
			t.Errorf("error message %q seems wrong", msg)
		}
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("wrong results (+got/-want):\n%s", diff)
	}
}

func TestSubscribeConduit(t *testing.T) {
	spy := apiresp(200, "create-eventsub-subscription-conduit.json")
	cl := Client{
		HTTP: &http.Client{Transport: spy},
	}
	tok := &oauth2.Token{AccessToken: "bocchi"}
	condition := map[string]string{"user_id": "1234"}
	conduit := "bfcfc993-26b1-b876-44d9-afe75a379dac"
	want := "26b1c993-bfcf-44d9-b876-379dacafe75a"
	got, err := SubscribeConduit(context.Background(), cl, tok, conduit, "user.update", "1", condition)
	if err != nil {
		t.Error(err)
	}
	if want != got {
		t.Errorf("wrong subscription id: want %q, got %q", want, got)
	}
}

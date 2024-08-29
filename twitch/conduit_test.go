package twitch

import (
	"context"
	"net/http"
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

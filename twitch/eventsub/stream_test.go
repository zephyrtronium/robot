package eventsub_test

import (
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/google/go-cmp/cmp"
	"github.com/zephyrtronium/robot/twitch/eventsub"
)

func TestStream(t *testing.T) {
	t.Run("online", func(t *testing.T) {
		evt := Testdata("stream.online.event.json")
		var got eventsub.Stream
		if err := json.Unmarshal([]byte(evt.Event), &got); err != nil {
			t.Errorf("couldn't unmarshal payload as stream: %v", err)
		}
		want := eventsub.Stream{
			ID:               "9001",
			Broadcaster:      "1337",
			BroadcasterLogin: "cool_user",
			BroadcasterName:  "Cool_User",
			Type:             "live",
			Started:          "2020-10-11T10:11:12.123Z",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong stream (+got/-want):\n%s", diff)
		}
	})
	t.Run("offline", func(t *testing.T) {
		evt := Testdata("stream.offline.event.json")
		var got eventsub.Stream
		if err := json.Unmarshal([]byte(evt.Event), &got); err != nil {
			t.Errorf("couldn't unmarshal payload as stream: %v", err)
		}
		want := eventsub.Stream{
			Broadcaster:      "1337",
			BroadcasterLogin: "cool_user",
			BroadcasterName:  "Cool_User",
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong stream (+got/-want):\n%s", diff)
		}
	})
}

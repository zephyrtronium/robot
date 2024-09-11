package eventsub_test

import (
	"embed"

	"github.com/go-json-experiment/json"

	"github.com/zephyrtronium/robot/twitch/eventsub"
)

//go:embed testdata/*.event.json
var testdata embed.FS

func Testdata(name string) *eventsub.Event {
	b, err := testdata.ReadFile("testdata/" + name)
	if err != nil {
		panic(err)
	}
	var r eventsub.Event
	if err := json.Unmarshal(b, &r); err != nil {
		panic(err)
	}
	return &r
}

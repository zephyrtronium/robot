package eventsub

import (
	"embed"
	"testing"

	"github.com/go-json-experiment/json"
)

//go:embed testdata/*.message.json
var messages embed.FS

func BenchmarkEventExtra(b *testing.B) {
	cases := []struct {
		name string
		file string
	}{
		{
			name: "keepalive",
			file: "keepalive.message.json",
		},
		{
			name: "notification-payload",
			file: "notification-payload.message.json",
		},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			p, err := messages.ReadFile("testdata/" + c.file)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				var m message
				if err := json.Unmarshal(p, &m); err != nil {
					b.Error(err)
				}
			}
		})
	}
}

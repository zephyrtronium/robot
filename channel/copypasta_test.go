package channel_test

import (
	"testing"
	"time"

	"github.com/zephyrtronium/robot/channel"
)

func TestMemeDetector(t *testing.T) {
	type meme struct {
		when int64
		who  string
		text string
		err  error
	}
	cases := []struct {
		name   string
		need   int
		within int64
		memes  []meme
	}{
		{
			name:   "simple",
			need:   2,
			within: 15,
			memes: []meme{
				{0, "bocchi", "madoka", channel.ErrNotCopypasta},
				{1, "ryou", "madoka", nil},
			},
		},
		{
			name:   "once",
			need:   2,
			within: 15,
			memes: []meme{
				{0, "bocchi", "madoka", channel.ErrNotCopypasta},
				{1, "ryou", "madoka", nil},
				{2, "nijika", "madoka", channel.ErrNotCopypasta},
				{3, "kita", "madoka", channel.ErrNotCopypasta},
			},
		},
		{
			name:   "different",
			need:   2,
			within: 15,
			memes: []meme{
				{0, "bocchi", "madoka", channel.ErrNotCopypasta},
				{1, "ryou", "homura", channel.ErrNotCopypasta},
				{2, "kita", "madoka", nil},
				{3, "nijika", "homura", nil},
			},
		},
		{
			name:   "needs",
			need:   4,
			within: 15,
			memes: []meme{
				{0, "bocchi", "madoka", channel.ErrNotCopypasta},
				{1, "ryou", "madoka", channel.ErrNotCopypasta},
				{2, "kita", "madoka", channel.ErrNotCopypasta},
				{3, "nijika", "madoka", nil},
			},
		},
		{
			name:   "time",
			need:   2,
			within: 15,
			memes: []meme{
				{0, "bocchi", "madoka", channel.ErrNotCopypasta},
				{20, "ryou", "madoka", channel.ErrNotCopypasta},
			},
		},
		{
			name:   "monotonic",
			need:   2,
			within: 15,
			memes: []meme{
				{0, "bocchi", "madoka", channel.ErrNotCopypasta},
				{20, "ryou", "homura", channel.ErrNotCopypasta},
				{1, "nijika", "madoka", channel.ErrNotCopypasta},
				{40, "kita", "madoka", channel.ErrNotCopypasta},
			},
		},
		{
			name:   "who",
			need:   2,
			within: 15,
			memes: []meme{
				{0, "bocchi", "madoka", channel.ErrNotCopypasta},
				{1, "bocchi", "madoka", channel.ErrNotCopypasta},
				{2, "bocchi", "madoka", channel.ErrNotCopypasta},
				{3, "ryou", "madoka", nil},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := channel.NewMemeDetector(c.need, time.Duration(c.within)*time.Millisecond)
			for _, m := range c.memes {
				err := d.Check(time.UnixMilli(m.when), m.who, m.text)
				if err != m.err {
					t.Errorf("wrong error for %+v: want %v, got %v", m, m.err, err)
				}
			}
		})
	}
}

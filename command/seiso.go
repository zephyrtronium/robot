package command

import (
	"context"
	"hash/maphash"
	"log/slog"
	"math/rand/v2"
	"time"

	"gitlab.com/zephyrtronium/pick"

	"github.com/zephyrtronium/robot/message"
)

var seisoSeed = maphash.MakeSeed()

func seisoScore(channel, user string, now time.Time) float64 {
	now = now.UTC()
	b := make([]byte, 0, 128)
	b = append(b, channel...)
	b = append(b, user...)
	b = now.AppendFormat(b, time.DateOnly)
	u := maphash.Bytes(seisoSeed, b)
	b = now.AppendFormat(b, "15") // hour only
	v := maphash.Bytes(seisoSeed, b)
	b = now.AppendFormat(b, "0405") // minute and second
	w := maphash.Bytes(seisoSeed, b)
	x := int64(u>>12+v>>14+w>>16) - int64(v<<50>>15)
	if x < 0 {
		x *= 10 // hehehe
	}
	return float64(x*100) / (1<<52 + 1<<48)
}

// Seiso calculates how "seiso" (Zoe tells me this means バニー姿 or "bunny form")
// the user is.
func Seiso(ctx context.Context, robo *Robot, call *Invocation) {
	if t := call.Channel.SilentTime(); call.Message.Time().Before(t) {
		robo.Log.InfoContext(ctx, "silent", slog.Time("until", call.Channel.SilentTime()))
		return
	}
	v, _ := call.Channel.Extra.Load(partnerKey{})
	e := call.Channel.Emotes.Pick(rand.Uint32())
	if v, _ := v.(*suitor); v != nil && v.who == call.Message.Sender.ID {
		m := パートナーの清楚.Pick(rand.Uint32())
		call.Channel.Message(ctx, message.Format("%s %s", m, e).AsReply(call.Message.ID))
		return
	}
	x := seisoScore(call.Channel.Name, call.Message.Sender.ID, call.Message.Time())
	m := 清楚.Pick(rand.Uint32())
	call.Channel.Message(ctx, message.Format("%s %.0f%s %s", m[0], x, m[1], e).AsReply(call.Message.ID))
}

var パートナーの清楚 = pick.New([]pick.Case[string]{
	{E: "Of course you're seiso, sweetie!", W: 10},
	// TODO(zeph): more
})

var 清楚 = pick.New([]pick.Case[[2]string]{
	{E: [2]string{"You're like", "% seiso"}, W: 10},
	{E: [2]string{"I think you're about", "% seiso"}, W: 10},
	{E: [2]string{"Like", "% seiso. Take that as you will."}, W: 2},
	{E: [2]string{"You are exactly", "% seiso"}, W: 10},
})

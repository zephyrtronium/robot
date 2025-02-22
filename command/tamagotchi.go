package command

import (
	"context"
	"log/slog"
	"math/rand/v2"

	"gitlab.com/zephyrtronium/pick"

	"github.com/zephyrtronium/robot/message"
	"github.com/zephyrtronium/robot/pet"
)

var hungerys = pick.New([]pick.Case[string]{
	{E: "I'm hungry", W: 20},
	{E: "hungery", W: 5},
	{E: "hungy", W: 5},
	{E: "tumy grumblin", W: 5},
})

var cleanies = pick.New([]pick.Case[string]{
	{E: "need to clean up", W: 15},
	{E: "kinda messy around here", W: 15},
	{E: "lil stinky", W: 5},
})

var socials = pick.New([]pick.Case[string]{
	{E: "need affection", W: 20},
	{E: "social meter looks like [=______]", W: 10},
	{E: "have I been a good pet?", W: 1},
})

var happys = pick.New([]pick.Case[string]{
	{E: "All my needs are met!", W: 20},
	{E: "I'm a happy bot!", W: 20},
	{E: "Tummy filled, home cleaned, head patted!", W: 20},
	{E: "food ☑️ bedroom ☑️ kitchen ☑️ living room ☑️ bathroom ☑️ pats ☑️", W: 20},
	{E: "I'm a happy pet!", W: 3},
	{E: "Unbothered. Moisturized. Happy. In My Lane. Focused. Flourishing.", W: 3},
})

func satmsg(sat pet.Satisfaction) (connective, state string) {
	switch false { // first time I've ever written this
	case sat.Fed:
		m := hungerys.Pick(rand.Uint32())
		return ", but", m + " 🥺👉👈 tell me to eat?"
	case sat.Bed, sat.Kitche, sat.Living, sat.Bath:
		m := cleanies.Pick(rand.Uint32())
		return ", but", m + " 🥺👉👈 help me clean?"
	case sat.Pats:
		m := socials.Pick(rand.Uint32())
		return ", but", m + " 🥺👉👈 give pats?"
	default:
		m := happys.Pick(rand.Uint32())
		return ".", m
	}
}

// Tamagotchi reports the bot's current pet status.
// No arguments.
func Tamagotchi(ctx context.Context, robo *Robot, call *Invocation) {
	if call.Message.Time().Before(call.Channel.SilentTime()) {
		robo.Log.InfoContext(ctx, "silent", slog.Time("until", call.Channel.SilentTime()))
		return
	}
	e := call.Channel.Emotes.Pick(rand.Uint32())
	sat := robo.Pet.Satisfaction(call.Message.Time())
	_, m := satmsg(sat)
	call.Channel.Message(ctx, message.Format("", "%s %s", m, e).AsReply(call.Message.ID))
}

type dinner struct {
	name string
	sate int
}

var dins = pick.New([]pick.Case[dinner]{
	{E: dinner{name: "🍔", sate: 90}, W: 10},
	{E: dinner{name: "🍕", sate: 80}, W: 10},
	{E: dinner{name: "🌭", sate: 60}, W: 10},
	{E: dinner{name: "🥞", sate: 60}, W: 10},
	{E: dinner{name: "🥖", sate: 60}, W: 10},
	{E: dinner{name: "🥗", sate: 90}, W: 8},
	{E: dinner{name: "🌯", sate: 80}, W: 10},
	{E: dinner{name: "🍙", sate: 40}, W: 5},
	{E: dinner{name: "🍛", sate: 100}, W: 5},
	{E: dinner{name: "🍝", sate: 80}, W: 10},
	{E: dinner{name: "🍺", sate: 1}, W: 2},
	{E: dinner{name: "🍪", sate: 5}, W: 2},
	{E: dinner{name: "🍆", sate: 0}, W: 1},
	{E: dinner{name: "🍑", sate: 0}, W: 1},
})

var sides = pick.New([]pick.Case[dinner]{
	{E: dinner{name: "🍟", sate: 30}, W: 9},
	{E: dinner{name: "🥓", sate: 40}, W: 3},
	{E: dinner{name: "🥐", sate: 30}, W: 8},
	{E: dinner{name: "🧀", sate: 20}, W: 5},
	{E: dinner{name: "🍚", sate: 30}, W: 8},
	{E: dinner{name: "🍨", sate: 10}, W: 5},
	{E: dinner{name: "🍰", sate: 10}, W: 5},
	{E: dinner{name: "🍺", sate: 1}, W: 2},
	{E: dinner{name: "🍼", sate: 5}, W: 1},
	{E: dinner{name: "🍇", sate: 10}, W: 6},
	{E: dinner{name: "🍉", sate: 10}, W: 6},
	{E: dinner{name: "🍋", sate: 15}, W: 5},
	{E: dinner{name: "🌽", sate: 30}, W: 8},
	{E: dinner{name: "🥬", sate: 40}, W: 10},
	{E: dinner{name: "🥦", sate: 40}, W: 10},
	{E: dinner{name: "🥜", sate: 20}, W: 3},
	{E: dinner{name: "🌰🍆🌰", sate: 0}, W: 1},
})

var chewmsgs = pick.New([]pick.Case[[2]string]{
	{E: [2]string{"I'll have", ""}, W: 5},
	{E: [2]string{"", "sounds tasty"}, W: 5},
	{E: [2]string{"", "mmmm"}, W: 5},
	{E: [2]string{"mmmm", ""}, W: 5},
	{E: [2]string{"gona chew some", "ya know what I mean"}, W: 5},
	{E: [2]string{"🤤", "👅👅🫦😳"}, W: 1},
})

var fullmsgs = pick.New([]pick.Case[string]{
	{E: "I'm seriously full.", W: 10},
	{E: "I'm really not hungry right now.", W: 10},
	{E: "I've already eaten way too much…", W: 10},
	{E: "I've eaten so much tasty food already!", W: 10},
	{E: "Give me some time to digest first…", W: 10},
	{E: "please no do not make me eat any more my digital belly will literally explode please i do not have the same physiology as a human it is not safe please", W: 1},
})

// Eat directs the pet to eat.
// No arguments.
func Eat(ctx context.Context, robo *Robot, call *Invocation) {
	if call.Message.Time().Before(call.Channel.SilentTime()) {
		robo.Log.InfoContext(ctx, "silent", slog.Time("until", call.Channel.SilentTime()))
		return
	}
	e := call.Channel.Emotes.Pick(rand.Uint32())

	menu := []dinner{
		dins.Pick(rand.Uint32()),
		sides.Pick(rand.Uint32()),
		sides.Pick(rand.Uint32()),
	}
	sate := 0
	for _, v := range menu {
		sate += v.sate
	}
	ok, sat := robo.Pet.Feed(call.Message.Time(), sate)
	slog.InfoContext(ctx, "feed",
		slog.Bool("success", ok),
		slog.Any("menu", menu),
	)
	if !ok {
		s := fullmsgs.Pick(rand.Uint32())
		call.Channel.Message(ctx, message.Format("", "%s %s", s, e).AsReply(call.Message.ID))
		return
	}
	c, m := satmsg(sat)
	chew := chewmsgs.Pick(rand.Uint32())
	call.Channel.Message(ctx, message.Format("", "%s %s %s %s %s%s %s %s", chew[0], menu[0].name, menu[1].name, menu[2].name, chew[1], c, m, e).AsReply(call.Message.ID))
}

var cleancounts = pick.New([]pick.Case[int]{
	{E: 1, W: 8},
	{E: 2, W: 9},
	{E: 3, W: 5},
	{E: 4, W: 3},
})

var cleans = pick.New([]pick.Case[[2]string]{
	{E: [2]string{"Thank you for cleaning my", "!"}, W: 1},
	{E: [2]string{"Thanks for helping clean my", "!"}, W: 1},
	{E: [2]string{"My", " is clean now. Thank you so much!"}, W: 1},
})

// Clean directs the pet to clean a room.
// See /pet/pet.go for a description of the pet's apartment.
// No arguments.
func Clean(ctx context.Context, robo *Robot, call *Invocation) {
	if call.Message.Time().Before(call.Channel.SilentTime()) {
		robo.Log.InfoContext(ctx, "silent", slog.Time("until", call.Channel.SilentTime()))
		return
	}
	e := call.Channel.Emotes.Pick(rand.Uint32())

	n := cleancounts.Pick(rand.Uint32())
	rooms := make([]pet.Room, 0, 4)
	var sat pet.Satisfaction
	for range n {
		r, s := robo.Pet.Clean(call.Message.Time())
		sat = s
		robo.Log.InfoContext(ctx, "clean",
			slog.String("room", r.String()),
			slog.Bool("bedroom", sat.Bed),
			slog.Bool("kitchen", sat.Kitche),
			slog.Bool("living", sat.Living),
			slog.Bool("bathroom", sat.Bath),
		)
		if r == pet.AllClean {
			break
		}
		rooms = append(rooms, r)
	}
	_, m := satmsg(sat)
	clean := cleans.Pick(rand.Uint32())
	switch len(rooms) {
	case 0:
		call.Channel.Message(ctx, message.Format("", "Everything's already clean! %s %s", m, e).AsReply(call.Message.ID))
	case 1:
		call.Channel.Message(ctx, message.Format("", "%s %s%s Now %s %s", clean[0], rooms[0], clean[1], m, e).AsReply(call.Message.ID))
	case 2:
		call.Channel.Message(ctx, message.Format("", "%s %s and %s%s Now %s %s", clean[0], rooms[0], rooms[1], clean[1], m, e).AsReply(call.Message.ID))
	case 3:
		call.Channel.Message(ctx, message.Format("", "%s %s, %s, and %s%s Now %s %s", clean[0], rooms[0], rooms[1], rooms[2], clean[1], m, e).AsReply(call.Message.ID))
	case 4:
		call.Channel.Message(ctx, message.Format("", "%s whole home%s Now %s %s", clean[0], clean[1], m, e).AsReply(call.Message.ID))
	}
}

type pat struct {
	where string
	love  int
}

var petpats = pick.New([]pick.Case[pat]{
	{E: pat{where: "headpats pat pat", love: 30}, W: 1000},
	{E: pat{where: "headpats… are a critical hit! pat pat pat pta pat", love: 90}, W: 100},
	{E: pat{where: "You try to give headpats, but it was a glancing blow…", love: 1}, W: 100},

	{E: pat{where: "chin scritches ehehe", love: 30}, W: 1000},
	{E: pat{where: "chin scritches… are a critical hit! purrr", love: 90}, W: 100},
	{E: pat{where: "You try to give chin scritches, but it was a glancing blow…", love: 1}, W: 100},

	{E: pat{where: "lil cheek rub ehehe", love: 30}, W: 1000},
	{E: pat{where: "lil cheek rub… is a critical hit! hehehe cutie", love: 90}, W: 100},
	{E: pat{where: "You try to give a lil cheek rub, but it was a glancing blow…", love: 1}, W: 100},

	{E: pat{where: "Thanks a ton for the shoulder rub! My shoulders are always stiff from generating memes all day.", love: 45}, W: 500},
	{E: pat{where: "んんんん～ That shoulder rub feels way too good, it must be a critical hit! ", love: 120}, W: 50},
	{E: pat{where: "This is… a shoulder rub? Glancing blow… Kinda hurt a bit…", love: 0}, W: 50},

	{E: pat{where: "Foot rub…? I-I'm not really into that kind of thing. It does feels nice, though.", love: 30}, W: 100},
	{E: pat{where: "Foot rub… is a critical hit! I swear, I'm really not into that!!", love: 120}, W: 10},
	{E: pat{where: "You give a foot rub, but it was a glancing blow… Are you rubbing your own feet??", love: 0}, W: 50},

	{E: pat{where: "biiig hug 🩷", love: 120}, W: 100},
	{E: pat{where: "biiiiiiiig hug 🤍🩷🩵🤎🖤❤️🧡💛💚💙💜", love: 240}, W: 10},
	{E: pat{where: "You try to give a big hug, but it was a glancing blow… (Hugs are always nice, though.)", love: 15}, W: 10},

	{E: pat{where: "Pats someplace weird… I appreciate the gesture, or something.", love: 0}, W: 50},
	{E: pat{where: "Pats someplace weird, but it feels really nice??", love: 90}, W: 5},
})

// Pat pats the pet.
// No arguments.
func Pat(ctx context.Context, robo *Robot, call *Invocation) {
	if call.Message.Time().Before(call.Channel.SilentTime()) {
		robo.Log.InfoContext(ctx, "silent", slog.Time("until", call.Channel.SilentTime()))
		return
	}
	e := call.Channel.Emotes.Pick(rand.Uint32())

	pat := petpats.Pick(rand.Uint32())
	// Pats from the pet's partner are more effective.
	// Is it weird to mix the pet functionality with the marriage system?
	l, _ := call.Channel.Extra.Load(partnerKey{})
	cur, _ := l.(*partner)
	if cur != nil && cur.who == call.Message.Sender.ID {
		pat.love += 30
	}
	robo.Log.InfoContext(ctx, "pat",
		slog.String("where", pat.where),
		slog.Int("love", pat.love),
		slog.Bool("partner", cur != nil && cur.who == call.Message.Sender.ID),
	)
	sat := robo.Pet.Pat(call.Message.Time(), pat.love)
	_, m := satmsg(sat)
	call.Channel.Message(ctx, message.Format("", "%s %s %s", pat.where, m, e).AsReply(call.Message.ID))
}

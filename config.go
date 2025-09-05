package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"gitlab.com/zephyrtronium/pick"
	"gitlab.com/zephyrtronium/tmi"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/sha3"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/zephyrtronium/robot/auth"
	"github.com/zephyrtronium/robot/brain/kvbrain"
	"github.com/zephyrtronium/robot/brain/sqlbrain"
	"github.com/zephyrtronium/robot/channel"
	"github.com/zephyrtronium/robot/message"
	"github.com/zephyrtronium/robot/privacy"
	"github.com/zephyrtronium/robot/spoken"
	"github.com/zephyrtronium/robot/twitch"
)

// Load loads Robot from a TOML configuration.
func Load(ctx context.Context, r io.Reader) (*Config, *toml.MetaData, error) {
	var cfg Config
	md, err := toml.NewDecoder(r).Decode(&cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't decode config: %w", err)
	}
	expandcfg(&cfg, func(s string) string {
		v, ok := os.LookupEnv(s)
		if !ok {
			slog.ErrorContext(ctx, "config refers to missing environment variable", slog.String("variable", s))
		}
		return v
	})
	return &cfg, &md, nil
}

// SetOwner sets owner metadata used in self-description commands.
func (robo *Robot) SetOwner(ownerName, ownerContact string) {
	robo.owner = ownerName
	robo.ownerContact = ownerContact
}

// loadSecrets loads Robot's fixed secret and initializes derived secrets.
func loadSecrets(file string) (*keys, error) {
	k, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("couldn't read secret key: %w", err)
	}
	uk := domainkey(make([]byte, 64), k, []byte("userhash"))
	tk := domainkey(make([]byte, auth.KeySize), k, []byte("oauth2.twitch"))
	r := &keys{
		userhash: uk,
		twitch:   (*[auth.KeySize]byte)(tk),
	}
	return r, nil
}

// SetSources opens the brain and privacy list wrappers around the respective
// databases. Use [loadDBs] to open the databases themselves from DSNs.
// Panics if both kv and sql are nil.
func (robo *Robot) SetSources(ctx context.Context, kv *badger.DB, sql, priv, spoke *sqlitex.Pool) error {
	var err error
	if sql == nil {
		if kv == nil {
			panic("robot: no brain")
		}
		robo.brain = kvbrain.New(kv)
	} else {
		robo.brain, err = sqlbrain.Open(ctx, sql)
	}
	if err != nil {
		return fmt.Errorf("couldn't open brain: %w", err)
	}
	robo.privacy, err = privacy.Open(ctx, priv)
	if err != nil {
		return fmt.Errorf("couldn't open privacy list: %w", err)
	}
	robo.spoken, err = spoken.Open(ctx, spoke)
	if err != nil {
		return fmt.Errorf("couldn't open spoken history: %w", err)
	}
	return nil
}

// InitTwitch initializes the Twitch and TMI clients and channel configuration.
func (robo *Robot) InitTwitch(ctx context.Context, cfg ClientCfg, secrets *keys, clientSecret string) error {
	cfg.secret = clientSecret
	cfg.endpoint = oauth2.Endpoint{
		DeviceAuthURL: "https://id.twitch.tv/oauth2/device",
		TokenURL:      "https://id.twitch.tv/oauth2/token",
	}
	send := make(chan *tmi.Message, 1)
	recv := make(chan *tmi.Message, 8) // 8 is enough for on-connect msgs
	client := &http.Client{Timeout: 30 * time.Second}
	robo.twitch = twitch.Client{HTTP: client, ID: cfg.CID}
	tmi, err := loadClient(
		cfg,
		send,
		recv,
		func(c oauth2.Config, s auth.Storage) auth.TokenSource {
			return auth.DeviceCodeFlow(c, s, client, deviceCodePrompt)
		},
		*secrets.twitch,
		"chat:read", "chat:edit",
	)
	if err != nil {
		return fmt.Errorf("couldn't load TMI client: %w", err)
	}
	robo.tmi = tmi
	// Validate the Twitch access token now to get our user ID and login.
	tok, err := robo.tmi.tokens.Token(ctx)
	if err != nil {
		return fmt.Errorf("couldn't obtain Twitch access token: %w", err)
	}
	for range 5 {
		val, err := twitch.Validate(ctx, robo.twitch.HTTP, tok)
		slog.InfoContext(ctx, "Twitch validation", slog.Any("response", val), slog.Any("err", err))
		switch {
		case err == nil:
			robo.tmi.name = val.Login
			robo.tmi.userID = val.UserID
			return nil

		case errors.Is(err, twitch.ErrNeedRefresh):
			tok, err = robo.tmi.tokens.Refresh(ctx, tok)
			if err != nil {
				return fmt.Errorf("couldn't refresh Twitch token: %w", err)
			}
			continue

		default:
			return fmt.Errorf("couldn't validate Twitch token: %w", err)
		}
	}
	return fmt.Errorf("gave up on validation attempts")
}

// InitTwitchUsers resolves Twitch usernames in the configuration to user IDs.
// It must be called after SetTMI.
func (robo *Robot) InitTwitchUsers(ctx context.Context, owner *Privilege, global []Privilege, channels map[string]*ChannelCfg) error {
	tok, err := robo.tmi.tokens.Token(ctx)
	if err != nil {
		return err
	}
	switch {
	case owner == nil:
		// Make it a fake pointer instead.
		owner = new(Privilege)
	default:
		for {
			r := []twitch.User{{ID: owner.ID, Login: owner.Name}}
			r, err := twitch.Users(ctx, robo.twitch, tok, r)
			switch {
			case err == nil: // do nothing
			case errors.Is(err, twitch.ErrNeedRefresh):
				tok, err = robo.tmi.tokens.Refresh(ctx, tok)
				if err != nil {
					return fmt.Errorf("couldn't refresh Twitch token: %w", err)
				}
				continue
			default:
				return fmt.Errorf("couldn't resolve owner info: %w", err)
			}
			*owner = Privilege{ID: r[0].ID, Name: r[0].Login, Level: "admin"}
			slog.InfoContext(ctx, "Twitch owner",
				slog.String("id", r[0].ID),
				slog.String("login", r[0].Login),
				slog.String("display", r[0].DisplayName),
			)
			break
		}
	}
	if owner.ID == "" {
		slog.WarnContext(ctx, "no owner information; continuing with owner commands disabled")
	}

	var in, out []twitch.User
	id := make(map[string][]*Privilege)
	login := make(map[string][]*Privilege)
	var mu sync.Mutex
	for i, p := range global {
		s := strings.ToLower(p.Name)
		in = append(in, twitch.User{ID: p.ID, Login: p.Name})
		id[s] = append(id[s], &global[i])
		login[s] = append(login[s], &global[i])
	}
	for _, ch := range channels {
		for i, p := range ch.Privileges {
			s := strings.ToLower(p.Name)
			in = append(in, twitch.User{ID: p.ID, Login: p.Name})
			id[s] = append(id[s], &ch.Privileges[i])
			login[s] = append(login[s], &ch.Privileges[i])
		}
	}
	group, ctx := errgroup.WithContext(ctx)
	for len(in) > 0 {
		l := in[:min(len(in), 100)]
		in = in[len(l):]
		group.Go(func() error {
			for {
				// TODO(zeph): rate limit
				l, err := twitch.Users(ctx, robo.twitch, tok, l)
				switch {
				case err == nil: // do nothing
				case errors.Is(err, twitch.ErrNeedRefresh):
					tok, err = robo.tmi.tokens.Refresh(ctx, tok)
					if err != nil {
						return fmt.Errorf("couldn't refresh Twitch token: %w", err)
					}
					continue
				default:
					return err
				}
				slog.InfoContext(ctx, "resolved users", slog.Int("count", len(l)))
				slog.DebugContext(ctx, "resolved users", slog.Any("users", l))
				mu.Lock()
				defer mu.Unlock()
				out = append(out, l...)
				return nil
			}
		})
	}
	if err := group.Wait(); err != nil {
		return fmt.Errorf("couldn't resolve config Twitch users: %w", err)
	}
	for _, u := range out {
		slog.DebugContext(ctx, "Twitch user",
			slog.String("id", u.ID),
			slog.String("login", u.Login),
			slog.String("display", u.DisplayName),
		)
		pp := id[u.ID]
		if pp == nil {
			pp = login[strings.ToLower(u.Login)]
			if pp == nil {
				slog.ErrorContext(ctx, "Twitch user for no one (continuing)",
					slog.String("id", u.ID),
					slog.String("login", u.Login),
					slog.String("display", u.DisplayName),
				)
				continue
			}
		}
		for _, p := range pp {
			p.ID = u.ID
			p.Name = u.Login
		}
	}
	return nil
}

func mergere(global, ch string) (*regexp.Regexp, error) {
	var re string
	switch {
	case global != "" && ch != "":
		re = "(" + global + ")|(" + ch + ")"
	case global != "":
		re = global
	case ch != "":
		re = ch
	default:
		re = "$^"
	}
	return regexp.Compile(re)
}

func (robo *Robot) SetDiscordChannels(global Global, discord Discord) error {
	for _, config := range discord.Configs {
		blk, err := mergere(global.Block, config.Block)
		if err != nil {
			return fmt.Errorf("bad global or channel block expression for discord.%s: %w", config.Server, err)
		}
		c := &channel.Channel{
			Name:      config.Server,
			Learn:     config.Learn,
			Send:      config.Send,
			Responses: config.Responses,
			Rate:      rate.NewLimiter(rate.Every(fseconds(config.Rate.Every)), config.Rate.Num),
			Block:     blk,
		}
		robo.channels.Store(config.Server, c)
	}
	return nil
}

// SetTwitchChannels initializes Twitch channel configuration.
// It must be called after SetTMI.
func (robo *Robot) SetTwitchChannels(ctx context.Context, global Global, channels map[string]*ChannelCfg) error {
	// TODO(zeph): we can convert this to a SetChannels, where it just adds the
	// channels for any given service
	for nm, ch := range channels {
		blk, err := mergere(global.Block, ch.Block)
		if err != nil {
			return fmt.Errorf("bad global or channel block expression for twitch.%s: %w", nm, err)
		}
		meme, err := mergere(global.Meme, ch.Meme)
		if err != nil {
			return fmt.Errorf("bad global or channel meme expression for twitch.%s: %w", nm, err)
		}
		emotes := pick.New(pick.FromMap(mergemaps(global.Emotes, ch.Emotes)))
		effects := pick.New(pick.FromMap(mergemaps(global.Effects, ch.Effects)))
		perms := make(map[string]channel.UserPerms)
		for _, p := range global.Privileges.Twitch {
			switch {
			case strings.EqualFold(p.Level, "ignore"):
				perms[p.ID] = channel.UserPerms{
					DisableCommands: true,
					DisableLearn:    true,
					DisableSpeak:    true,
					DisableMemes:    true,
				}
			case strings.EqualFold(p.Level, "selfbot"):
				perms[p.ID] = channel.UserPerms{DisableLearn: true}
			case strings.EqualFold(p.Level, "moderator"):
				perms[p.ID] = channel.UserPerms{Moderator: true}
			}
		}
		for _, p := range ch.Privileges {
			switch {
			case strings.EqualFold(p.Level, "ignore"):
				perms[p.ID] = channel.UserPerms{
					DisableCommands: true,
					DisableLearn:    true,
					DisableSpeak:    true,
					DisableMemes:    true,
				}
			case strings.EqualFold(p.Level, "selfbot"):
				perms[p.ID] = channel.UserPerms{DisableLearn: true}
			case strings.EqualFold(p.Level, "moderator"):
				perms[p.ID] = channel.UserPerms{Moderator: true}
			}
		}
		for _, p := range ch.Channels {
			v := &channel.Channel{
				Name:        p,
				Learn:       ch.Learn,
				Send:        ch.Send,
				Block:       blk,
				Meme:        meme,
				Responses:   ch.Responses,
				Rate:        rate.NewLimiter(rate.Every(fseconds(ch.Rate.Every)), ch.Rate.Num),
				Permissions: perms,
				Memery:      channel.NewMemeDetector(ch.Copypasta.Need, fseconds(ch.Copypasta.Within)),
				Emotes:      emotes,
				Effects:     effects,
			}
			v.Message = func(ctx context.Context, msg message.Sent) {
				if msg.To == "" {
					msg.To = v.Name
				}
				robo.sendTMI(ctx, robo.tmi.send, msg)
			}
			robo.channels.Store(p, v)
		}
	}
	return nil
}

func loadDBs(ctx context.Context, cfg DBCfg) (kv *badger.DB, sql, priv, spoke *sqlitex.Pool, err error) {
	if cfg.KVBrain != "" && cfg.SQLBrain != "" {
		return nil, nil, nil, nil, fmt.Errorf("multiple brain backends requested; use exactly one")
	}
	if cfg.KVBrain == "" && cfg.SQLBrain == "" {
		return nil, nil, nil, nil, fmt.Errorf("no brain backends requested; use exactly one")
	}

	if cfg.KVBrain != "" {
		slog.DebugContext(ctx, "using kvbrain", slog.String("path", cfg.KVBrain), slog.String("flags", cfg.KVFlag))
		opts := badger.DefaultOptions(cfg.KVBrain)
		// TODO(zeph): logger?
		opts = opts.WithLogger(nil)
		opts = opts.WithCompression(options.None)
		opts = opts.WithBloomFalsePositive(0)
		kv, err = badger.Open(opts.FromSuperFlag(cfg.KVFlag))
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("couldn't open kvbrain db: %w", err)
		}
	}
	if cfg.SQLBrain != "" {
		slog.DebugContext(ctx, "using sqlbrain", slog.String("path", cfg.SQLBrain))
		sql, err = sqlitex.NewPool(cfg.SQLBrain, sqlitex.PoolOptions{PrepareConn: sqlbrain.RecommendedPrep})
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("couldn't open sqlbrain db: %w", err)
		}
	}

	switch cfg.Privacy {
	case cfg.SQLBrain:
		slog.DebugContext(ctx, "privacy db shared with sqlbrain")
		priv = sql
	default:
		slog.DebugContext(ctx, "privacy db", slog.String("path", cfg.Privacy))
		priv, err = sqlitex.NewPool(cfg.Privacy, sqlitex.PoolOptions{})
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("couldn't open privacy db: %w", err)
		}
	}

	switch cfg.Spoken {
	case cfg.SQLBrain:
		slog.DebugContext(ctx, "spoken history db shared with sqlbrain")
		spoke = sql
	case cfg.Privacy:
		slog.DebugContext(ctx, "spoken history db shared with privacy db")
		spoke = priv
	default:
		slog.DebugContext(ctx, "spoken history db", slog.String("path", cfg.Spoken))
		spoke, err = sqlitex.NewPool(cfg.Spoken, sqlitex.PoolOptions{})
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("couldn't open spoken history db: %w", err)
		}
	}

	return kv, sql, priv, spoke, nil
}

func mergemaps(ms ...map[string]int) map[string]int {
	u := make(map[string]int)
	for _, m := range ms {
		for k, v := range m {
			u[k] += v
		}
	}
	return u
}

func fseconds(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
}

func loadClientSecret(file string) (string, error) {
	secret, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("couldn't read client secret: %w", err)
	}
	secret = bytes.TrimSuffix(bytes.TrimSuffix(secret, []byte{'\n'}), []byte{'\r'})
	return string(secret), nil
}

// loadClient loads client configuration from unmarshaled TOML.
func loadClient[Send, Receive any](
	t ClientCfg,
	send chan Send,
	recv chan Receive,
	tokens func(oauth2.Config, auth.Storage) auth.TokenSource,
	key [auth.KeySize]byte,
	scopes ...string,
) (*client[Send, Receive], error) {
	stor, err := auth.NewFileAt(t.TokenFile, key)
	if err != nil {
		return nil, fmt.Errorf("couldn't use refresh token storage: %w", err)
	}
	cfg := oauth2.Config{
		ClientID:     t.CID,
		ClientSecret: t.secret,
		Endpoint:     t.endpoint,
		RedirectURL:  t.RedirectURL,
		Scopes:       scopes,
	}
	return &client[Send, Receive]{
		send:     send,
		recv:     recv,
		clientID: t.CID,
		owner:    t.Owner.ID,
		rate:     rate.NewLimiter(rate.Every(fseconds(t.Rate.Every)), t.Rate.Num),
		tokens:   tokens(cfg, stor),
	}, nil
}

type keys struct {
	// userhash is the hasher for userhashes.
	userhash []byte
	// twitch is the key for Twitch OAuth2 token storage.
	twitch *[auth.KeySize]byte
}

// domainkey fills o with a key derived from k for the given domain. Panics if
// a key cannot be expanded.
func domainkey(o, k, domain []byte) []byte {
	kr := hkdf.Expand(sha3.New224, k, domain)
	if _, err := io.ReadFull(kr, o); err != nil {
		panic(err)
	}
	return o
}

// Config is the marshaled structure of Robot's configuration.
type Config struct {
	// SecretFile is the path to a file containing a secret key used to encrypt
	// durable secrets like OAuth2 refresh tokens as well as to create
	// userhashes.
	SecretFile string `toml:"secret"`
	// Owner is the table of metadata about the owner.
	Owner Owner `toml:"owner"`
	// DB is the table of database connection strings.
	DB DBCfg `toml:"db"`
	// HTTP is the table of HTTP API settings.
	HTTP APICfg `toml:"http"`
	// Global is the table of global settings.
	Global Global `toml:"global"`
	// TMI is the configuration for connecting to Twitch chat.
	TMI ClientCfg `toml:"tmi"`
	// Twitch is the set of channel configurations for twitch. Each key
	// represents a group of one or more channels sharing a config.
	Twitch map[string]*ChannelCfg `toml:"twitch"`
	// Discord is the discord client configuration options
	Discord Discord `toml:"discord"`
}

type Discord struct {
	// Token is the discord bot token
	Token string `toml:"token"`
	// Configs are the discord server configs
	Configs []DiscordCfg `toml:"configs"`
}

type DiscordCfg struct {
	// Server is the discord server ID for this configuration
	Server string `toml:"server"`
	// Learn is the tag used for learning from these channels.
	Learn string `toml:"learn"`
	// Send is the tag used for generating messages for these channels.
	Send string `toml:"send"`
	// Responses is the probability of generating a random message when
	// a non-command message is received.
	Responses float64 `toml:"responses"`
	// Rate is the rate limit for interactions.
	Rate Rate `toml:"rate"`
	// Block is a regular expression of messages to ignore.
	Block string `toml:"block"`
}

// ChannelCfg is the configuration for a channel.
type ChannelCfg struct {
	// Channels is the list of channels using this config.
	Channels []string `toml:"channels"`
	// Learn is the tag used for learning from these channels.
	Learn string `toml:"learn"`
	// Send is the tag used for generating messages for these channels.
	Send string `toml:"send"`
	// Block is a regular expression of messages to ignore.
	Block string `toml:"block"`
	// Responses is the probability of generating a random message when
	// a non-command message is received.
	Responses float64 `toml:"responses"`
	// Rate is the rate limit for interactions.
	Rate Rate `toml:"rate"`
	// Copypasta is the configuration for copypasta.
	Copypasta Copypasta `toml:"copypasta"`
	// Meme is a regular expression of messages to allow to be copypasta even
	// if matched by this channel's or the global Block.
	Meme string `toml:"meme"`
	// Emotes is the emotes and their weights for the channel.
	Emotes map[string]int `toml:"emotes"`
	// Effects is the effects and their weights for the channel.
	Effects map[string]int `toml:"effects"`
	// Privileges is the user access controls for the channel.
	Privileges []Privilege `toml:"privileges"`
}

// Global is the configuration for globally applied options.
type Global struct {
	// Block is a regular expression of messages to ignore everywhere.
	Block string `toml:"block"`
	// Meme is a regular expression of messages to allow to be copypasta even
	// if matched by Block.
	Meme string `toml:"meme"`
	// Emotes is the emotes and their weights to use everywhere.
	Emotes map[string]int `toml:"emotes"`
	// Effects is the effects and their weights to use everywhere.
	Effects map[string]int `toml:"effects"`
	// Privileges is the user access controls across entire services.
	Privileges GlobalPrivs `toml:"privileges"`
}

// GlobalPrivs is the configuration for privileges across entire services.
type GlobalPrivs struct {
	Twitch []Privilege `toml:"twitch"`
}

// Owner is metadata about the bot owner.
type Owner struct {
	// Name is the name of the owner. It does not need to be a username.
	Name string `toml:"name"`
	// Contact describes owner contact information.
	Contact string `toml:"contact"`
}

// ClientCfg is the configuration for connecting to an OAuth2 interface.
type ClientCfg struct {
	// CID is the client ID.
	CID string `toml:"cid"`
	// SecretFile is the path to a file containing the client secret.
	SecretFile string `toml:"secret"`
	// RedirectURL is the redirect URL for OAuth2 flows. For clients that don't
	// use authorization code grant flow, it may be unused but still must match
	// the configuration on the platform.
	RedirectURL string `toml:"redirect"`
	// TokenFile is the path to a file in which the bot will persist its OAuth2
	// refresh token. It is encrypted with a key derived from the Config.Secret
	// key.
	TokenFile string `toml:"token"`
	// Owner is the user ID of the owner. The interpretation of this is
	// domain-specific.
	Owner Privilege `toml:"owner"`
	// Rate is the global rate limit for this client.
	Rate Rate `toml:"rate"`

	secret   string          `toml:"-"`
	endpoint oauth2.Endpoint `toml:"-"`
}

type Privilege struct {
	// ID is the user ID.
	ID string `toml:"id"`
	// Name is the user name or login name (not display name).
	// If both ID and Name are provided, Name is generally ignored.
	Name string `toml:"name"`
	// Level is the access level granted to the user.
	// Valid values are the empty string as the default capability,
	// "ignore" to disable access to all commands including prompting,
	// or "moderator" to enable access to moderation commands.
	Level string `toml:"level"`
}

// DBCfg is the configuration of databases.
type DBCfg struct {
	SQLBrain string `toml:"sqlbrain"`
	KVBrain  string `toml:"kvbrain"`
	KVFlag   string `toml:"kvflag"`
	Privacy  string `toml:"privacy"`
	Spoken   string `toml:"spoken"`
}

// APICfg is the configuration of the HTTP API.
type APICfg struct {
	// Listen is the address and port on which to listen.
	Listen string `toml:"listen"`
}

// Rate is a rate limit configuration.
type Rate struct {
	Every float64 `toml:"every"`
	Num   int     `toml:"num"`
}

// Copypasta is a copypasta configuration.
type Copypasta struct {
	Need   int     `toml:"need"`
	Within float64 `toml:"within"`
}

func expandcfg(cfg *Config, expand func(s string) string) {
	fields := []*string{
		&cfg.SecretFile,
		&cfg.Owner.Name,
		&cfg.Owner.Contact,
		&cfg.DB.SQLBrain,
		&cfg.DB.KVBrain,
		&cfg.DB.KVFlag,
		&cfg.DB.Privacy,
		&cfg.DB.Spoken,
		&cfg.HTTP.Listen,
		&cfg.TMI.CID,
		&cfg.TMI.SecretFile,
		&cfg.TMI.TokenFile,
		&cfg.TMI.Owner.Name,
		&cfg.TMI.Owner.ID,
		&cfg.Discord.Token,
	}
	for _, f := range fields {
		*f = os.Expand(*f, expand)
	}
	for _, v := range cfg.Twitch {
		for i, s := range v.Channels {
			v.Channels[i] = os.Expand(s, expand)
		}
		v.Learn = os.Expand(v.Learn, expand)
		v.Send = os.Expand(v.Send, expand)
	}
}

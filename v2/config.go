package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/go-chi/chi/v5"
	"gitlab.com/zephyrtronium/sq"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/sha3"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
	"golang.org/x/time/rate"

	"github.com/zephyrtronium/robot/v2/auth"
	"github.com/zephyrtronium/robot/v2/brain/sqlbrain"
	"github.com/zephyrtronium/robot/v2/brain/userhash"
	"github.com/zephyrtronium/robot/v2/channel"
	"github.com/zephyrtronium/robot/v2/distro"
	"github.com/zephyrtronium/robot/v2/privacy"
)

// Load loads Robot from a TOML configuration.
func Load(ctx context.Context, r io.Reader) (*Robot, error) {
	var cfg Config
	var robo Robot
	md, err := toml.NewDecoder(r).Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("couldn't decode config: %w", err)
	}
	expandcfg(&cfg, os.Getenv)

	robo.secrets, err = readkeys(cfg.SecretFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't read keys: %w", err)
	}

	br, pr, err := loadDBs(ctx, cfg.DB)
	if err != nil {
		return nil, err
	}
	robo.brain, err = sqlbrain.Open(ctx, br)
	if err != nil {
		return nil, fmt.Errorf("couldn't open brain: %w", err)
	}
	robo.privacy, err = privacy.Open(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("couldn't open privacy list: %w", err)
	}

	// TODO(zeph): real url
	baseURL := "http://" + cfg.HTTP.Address
	if strings.HasPrefix(cfg.HTTP.Address, ":") {
		baseURL = "http://localhost" + cfg.HTTP.Address
	}
	rtr := chi.NewRouter()
	// TODO(zeph): other routes

	if md.IsDefined("tmi") {
		cfg.TMI.endpoint = twitch.Endpoint
		cfg.TMI.redir = baseURL + "/login/twitch/callback"
		cfg.TMI.landing = baseURL
		robo.tmi, err = loadClient(ctx, cfg.TMI, *robo.secrets.twitch, "chat:read", "chat:edit")
		if err != nil {
			return nil, fmt.Errorf("couldn't load TMI config: %w", err)
		}
		rtr.Get("/login/twitch", robo.tmi.token.Login)
		rtr.Get("/login/twitch/callback", robo.tmi.token.Callback)
	}
	robo.http = http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           rtr,
		ReadHeaderTimeout: 5 * time.Second,
	}

	robo.channels = make(map[string]*channel.Channel)
	for nm, ch := range cfg.Twitch {
		blk, err := regexp.Compile("(" + cfg.Global.Block + ")|(" + ch.Block + ")")
		if err != nil {
			return nil, fmt.Errorf("bad global or channel block expression for twitch.%s: %w", nm, err)
		}
		emotes := distro.New(distro.FromMap(mergemaps(cfg.Global.Emotes, ch.Emotes)))
		var ign, mod map[string]bool
		for u, p := range ch.Privileges {
			// TODO(zeph): Users may be listed in the TOML as usernames or as
			// user IDs, but the maps we give back should be only IDs.
			switch {
			case strings.EqualFold(p, "ignore"):
				if ign == nil {
					ign = make(map[string]bool)
				}
				ign[u] = true
			case strings.EqualFold(p, "moderator"):
				if mod == nil {
					mod = make(map[string]bool)
				}
				mod[u] = true
			}
		}
		for _, p := range ch.Channels {
			v := channel.Channel{
				Name:      p,
				Learn:     ch.Learn,
				Send:      ch.Send,
				Block:     blk,
				Responses: ch.Responses,
				Rate:      rate.NewLimiter(rate.Every(fseconds(ch.Rate.Every)), ch.Rate.Num),
				Ignore:    ign,
				Mod:       mod,
				Memery:    channel.NewMemeDetector(ch.Copypasta.Need, fseconds(ch.Copypasta.Within)),
				Emotes:    emotes,
			}
			robo.channels[p] = &v
		}
	}

	robo.works = make(chan chan func(context.Context), runtime.GOMAXPROCS(0))
	robo.owner = cfg.Owner.Name
	robo.ownerContact = cfg.Owner.Contact

	return &robo, nil
}

// Init initializes the databases in a Robot TOML configuration.
func Init(ctx context.Context, r io.Reader, order int) error {
	if order <= 0 {
		return errors.New("order must be positive")
	}
	var cfg Config
	b, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("couldn't read config: %w", err)
	}
	if err := toml.Unmarshal(b, &cfg); err != nil {
		return fmt.Errorf("couldn't unmarshal config: %w", err)
	}
	expandcfg(&cfg, os.Getenv)
	brain, priv, err := loadDBs(ctx, cfg.DB)
	if err != nil {
		return err
	}
	if err := sqlbrain.Create(ctx, brain, order); err != nil {
		return fmt.Errorf("couldn't initialize brain: %w", err)
	}
	if err := privacy.Init(ctx, priv); err != nil {
		return fmt.Errorf("couldn't initialize privacy list: %w", err)
	}
	return nil
}

func loadDBs(ctx context.Context, cfg DBCfg) (brain, priv *sq.DB, err error) {
	brain, err = sq.Open("sqlite3", cfg.Brain)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't open brain db: %w", err)
	}
	if err := brain.Ping(ctx); err != nil {
		return nil, nil, fmt.Errorf("couldn't connect to brain db: %w", err)
	}

	if cfg.Privacy == cfg.Brain {
		priv = brain
		return brain, priv, nil
	}

	priv, err = sq.Open("sqlite3", cfg.Privacy)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't open privacy db: %w", err)
	}
	if err := priv.Ping(ctx); err != nil {
		return nil, nil, fmt.Errorf("couldn't connect to privacy db: %w", err)
	}

	return brain, priv, nil
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

// client is the settings for OAuth2 and related elements.
type client struct {
	// me is the bot's username. The interpretation of this is domain-specific.
	me string
	// owner is the user ID of the owner. The interpretation of this is
	// domain-specific.
	owner string
	// rate is the global rate limiter for this client.
	rate *rate.Limiter
	// token is the OAuth2 token.
	token *auth.Token
}

// loadClient loads client configuration from unmarshaled TOML.
func loadClient(ctx context.Context, t ClientCfg, key [auth.KeySize]byte, scopes ...string) (*client, error) {
	secret, err := os.ReadFile(t.SecretFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't read client secret: %w", err)
	}
	stor, err := auth.NewFileAt(t.TokenFile, key)
	if err != nil {
		return nil, fmt.Errorf("couldn't use refresh token storage: %w", err)
	}
	cfg := auth.Config{
		App: oauth2.Config{
			ClientID:     t.CID,
			ClientSecret: string(secret),
			Endpoint:     t.endpoint,
			RedirectURL:  t.redir,
			Scopes:       scopes,
		},
		Client: http.Client{
			Timeout: 30 * time.Second,
		},
		Landing: t.landing,
	}
	tok, err := auth.New(ctx, stor, cfg)
	if err != nil {
		return nil, fmt.Errorf("couldn't create token: %w", err)
	}
	return &client{
		me:    t.User,
		owner: t.Owner,
		rate:  rate.NewLimiter(rate.Every(fseconds(t.Rate.Every)), t.Rate.Num),
		token: tok,
	}, nil
}

type keys struct {
	// userhash is the hasher for userhashes.
	userhash userhash.Hasher
	// twitch is the key for Twitch OAuth2 token storage.
	twitch *[auth.KeySize]byte
}

// readkeys creates keys for userhashes and encryption from the base key at a
// given file.
func readkeys(file string) (*keys, error) {
	k, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	uk := domainkey(make([]byte, 64), k, []byte("userhash"))
	tk := domainkey(make([]byte, auth.KeySize), k, []byte("oauth2.twitch"))
	keys := keys{
		userhash: userhash.New(uk),
		twitch:   (*[32]byte)(tk),
	}
	return &keys, nil
}

// domainkey fills o with a key derived from k for the given domain. Panics if
// a key cannot be expanded.
func domainkey(o, k, domain []byte) []byte {
	kr := hkdf.Expand(sha3.New224, k, []byte(domain))
	if _, err := kr.Read(o); err != nil {
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
	// HTTP is the table of HTTP server settings.
	HTTP HTTPCfg `toml:"http"`
	// Owner is the table of metadata about the owner.
	Owner Owner `toml:"owner"`
	// DB is the table of database connection strings.
	DB DBCfg `toml:"db"`
	// Global is the table of global settings.
	Global Global `toml:"global"`
	// TMI is the configuration for connecting to Twitch chat.
	TMI ClientCfg `toml:"tmi"`
	// Twitch is the set of channel configurations for twitch. Each key
	// represents a group of one or more channels sharing a config.
	Twitch map[string]*ChannelCfg `toml:"twitch"`
}

// HTTPCfg is the configuration for the bot's HTTP server.
type HTTPCfg struct {
	// Address is the domain and port at which to bind the server.
	Address string `toml:"address"`
	// TODO(zeph): HTTPS support
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
	// Emotes is the emotes and their weights for the channel.
	Emotes map[string]int `toml:"emotes"`
	// Privileges is the user access controls for the channel.
	Privileges map[string]string `toml:"privileges"`
}

// Global is the configuration for globally applied options.
type Global struct {
	// Block is a regular expression of messages to ignore everywhere.
	Block string `toml:"block"`
	// Emotes is the emotes and their weights to use everywhere.
	Emotes map[string]int `toml:"emotes"`
}

// Owner is metadata about the bot owner.
type Owner struct {
	// Name is the username of the owner.
	Name string `toml:"name"`
	// Contact describes owner contact information.
	Contact string `toml:"contact"`
}

// ClientCfg is the configuration for connecting to an OAuth2 interface.
type ClientCfg struct {
	// User is robot's username. The interpretation of this is domain-specific.
	// On TMI, it is used to connect and to detect commands.
	User string `toml:"user"`
	// CID is the client ID.
	CID string `toml:"cid"`
	// SecretFile is the path to a file containing the client secret.
	SecretFile string `toml:"secret"`
	// TokenFile is the path to a file in which the bot will persist its OAuth2
	// refresh token. It is encrypted with a key derived from the Config.Secret
	// key.
	TokenFile string `toml:"token"`
	// Owner is the user ID of the owner. The interpretation of this is
	// domain-specific.
	Owner string `toml:"owner"`
	// Rate is the global rate limit for this client.
	Rate Rate `toml:"rate"`

	endpoint oauth2.Endpoint `toml:"-"`
	redir    string          `toml:"-"`
	landing  string          `toml:"-"`
}

// DBCfg is the configuration of databases.
type DBCfg struct {
	Brain   string `toml:"brain"`
	Privacy string `toml:"privacy"`
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
		&cfg.HTTP.Address,
		&cfg.Owner.Name,
		&cfg.Owner.Contact,
		&cfg.DB.Brain,
		&cfg.DB.Privacy,
		&cfg.TMI.User,
		&cfg.TMI.CID,
		&cfg.TMI.SecretFile,
		&cfg.TMI.TokenFile,
		&cfg.TMI.Owner,
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

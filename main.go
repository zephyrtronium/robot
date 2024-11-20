package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v3"
	"golang.org/x/sync/errgroup"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/zephyrtronium/robot/brain"
	"github.com/zephyrtronium/robot/brain/kvbrain"
	"github.com/zephyrtronium/robot/brain/sqlbrain"
	"github.com/zephyrtronium/robot/metrics"
	"github.com/zephyrtronium/robot/userhash"
)

var app = cli.Command{
	Name:  "robot",
	Usage: "Markov chain chat bot",

	Flags: []cli.Flag{
		&flagConfig,
		&flagLog,
		&flagLogFormat,
	},
	Commands: []*cli.Command{
		{
			Name:    "speak",
			Aliases: []string{"talk", "generate", "say"},
			Usage:   "Generate messages without serving",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "tag",
					Usage:    "Tag from which to generate messages",
					Required: true,
				},
				&cli.IntFlag{
					Name:  "n",
					Usage: "Number of messages to generate",
					Value: 10,
				},
				&cli.StringFlag{
					Name:  "prompt",
					Usage: "Prompt to use for all generated messages",
				},
				&cli.BoolFlag{
					Name:  "trace",
					Usage: "Print ID traces with messages",
				},
			},
			Action: cliSpeak,
		},
		{
			Name:  "ancient",
			Usage: "Import messages from a v0.1.0 Robot database",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "db",
					Usage:    "Ancient version database",
					Required: true,
				},
			},
			Action: cliAncient,
		},
	},
	Action: cliRun,

	Authors: []any{
		"Branden J Brown  @zephyrtronium",
	},
	Copyright: "Copyright 2024 Branden J Brown",
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		<-ctx.Done()
		stop()
	}()
	err := app.Run(ctx, os.Args)
	if err != nil {
		fmt.Println(err)
	}
}

func cliRun(ctx context.Context, cmd *cli.Command) error {
	slog.SetDefault(loggerFromFlags(cmd))
	r, err := os.Open(cmd.String("config"))
	if err != nil {
		return fmt.Errorf("couldn't open config file: %w", err)
	}
	cfg, md, err := Load(ctx, r)
	if err != nil {
		return fmt.Errorf("couldn't load config: %w", err)
	}
	r.Close()

	secrets, err := loadSecrets(cfg.SecretFile)
	if err != nil {
		return err
	}
	robo := New(secrets.userhash, runtime.GOMAXPROCS(0))
	robo.SetOwner(cfg.Owner.Name, cfg.Owner.Contact)
	kv, sql, priv, spoke, err := loadDBs(ctx, cfg.DB)
	if err != nil {
		return err
	}
	if err := robo.SetSources(ctx, kv, sql, priv, spoke); err != nil {
		return err
	}

	if md.IsDefined("tmi") {
		secret, err := loadClientSecret(cfg.TMI.SecretFile)
		if err != nil {
			return err
		}
		if err := robo.InitTwitch(ctx, cfg.TMI, secrets, secret); err != nil {
			return err
		}
		if err := robo.InitTwitchUsers(ctx, &cfg.TMI.Owner, cfg.Global.Privileges.Twitch, cfg.Twitch); err != nil {
			return err
		}
		if err := robo.SetTwitchChannels(ctx, cfg.Global, cfg.Twitch); err != nil {
			return err
		}
	}

	return robo.Run(ctx, cfg.HTTP.Listen)
}

func cliSpeak(ctx context.Context, cmd *cli.Command) error {
	slog.SetDefault(loggerFromFlags(cmd))
	r, err := os.Open(cmd.String("config"))
	if err != nil {
		return fmt.Errorf("couldn't open config file: %w", err)
	}
	cfg, _, err := Load(ctx, r)
	if err != nil {
		return fmt.Errorf("couldn't load config: %w", err)
	}
	r.Close()
	kv, sql, _, _, err := loadDBs(ctx, cfg.DB)
	if err != nil {
		return err
	}
	var br brain.Brain
	if sql == nil {
		if kv == nil {
			panic("robot: no brain")
		}
		br = kvbrain.New(kv)
		defer kv.Close()
	} else {
		br, err = sqlbrain.Open(ctx, sql)
		defer sql.Close()
	}
	if err != nil {
		return fmt.Errorf("couldn't open brain: %w", err)
	}
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(runtime.GOMAXPROCS(0))
	tag := cmd.String("tag")
	trace := cmd.Bool("trace")
	prompt := cmd.String("prompt")
	for range cmd.Int("n") {
		group.Go(func() error {
			m, tr, err := brain.Speak(ctx, br, tag, prompt)
			if err != nil {
				return err
			}
			a := []any{m}
			if trace {
				a = append(a, tr)
			}
			fmt.Println(a...)
			return nil
		})
	}
	return group.Wait()
}

func cliAncient(ctx context.Context, cmd *cli.Command) error {
	slog.SetDefault(loggerFromFlags(cmd))
	r, err := os.Open(cmd.String("config"))
	if err != nil {
		return fmt.Errorf("couldn't open config file: %w", err)
	}
	cfg, _, err := Load(ctx, r)
	if err != nil {
		return fmt.Errorf("couldn't load config: %w", err)
	}
	r.Close()
	kv, sql, _, _, err := loadDBs(ctx, cfg.DB)
	if err != nil {
		return err
	}
	var br brain.Brain
	if sql == nil {
		if kv == nil {
			panic("robot: no brain")
		}
		br = kvbrain.New(kv)
		defer kv.Close()
	} else {
		conn, _ := sql.Take(ctx)
		sqlitex.ExecuteScript(conn, `PRAGMA synchronous=OFF; PRAGMA journal_mode=OFF`, nil)
		sql.Put(conn)
		br, err = sqlbrain.Open(ctx, sql)
		defer sql.Close()
	}
	if err != nil {
		return fmt.Errorf("couldn't open brain: %w", err)
	}
	file := cmd.String("db")
	conn, order, err := ancientOpen(file)
	if err != nil {
		return fmt.Errorf("couldn't open ancient db: %w", err)
	}
	slog.InfoContext(ctx, "importing", slog.String("file", file), slog.Int("order", order))
	var n int64
	var toks []string
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for msg, err := range ancientMessages(conn, order) {
		n++
		if err != nil {
			slog.ErrorContext(ctx, "error getting message", slog.Any("err", err))
			return err
		}
		id := fmt.Sprintf("import:%s:%d", file, n)
		toks = brain.Tokens(toks[:0], msg.text)
		slog.DebugContext(ctx, "learn", slog.String("tag", msg.tag), slog.String("text", msg.text))
		if err := brain.Learn(ctx, br, msg.tag, id, userhash.Hash{}, time.Now(), toks); err != nil {
			slog.ErrorContext(ctx, "error learning message", slog.Any("err", err))
			return err
		}
		select {
		case <-t.C:
			slog.InfoContext(ctx, "imported", slog.Int64("n", n))
		default: // do nothing
		}
	}
	return nil
}

var (
	flagConfig = cli.StringFlag{
		Name:       "config",
		Required:   true,
		Usage:      "TOML config file",
		Persistent: true,
		Action: func(ctx context.Context, cmd *cli.Command, s string) error {
			i, err := os.Stat(s)
			if err != nil {
				return err
			}
			if !i.Mode().IsRegular() {
				return errors.New("config must be a regular file")
			}
			return nil
		},
	}

	flagLog = cli.StringFlag{
		Name:       "log",
		Usage:      "Logging level, one of debug, info, warn, error",
		Value:      "info",
		Persistent: true,
		Action: func(ctx context.Context, c *cli.Command, s string) error {
			var l slog.Level
			return l.UnmarshalText([]byte(s))
		},
	}

	flagLogFormat = cli.StringFlag{
		Name:       "log-format",
		Usage:      "Logging format, either text or json",
		Value:      "text",
		Persistent: true,
		Action: func(ctx context.Context, c *cli.Command, s string) error {
			switch strings.ToLower(s) {
			case "text", "json":
				return nil
			default:
				return errors.New("unknown logging format")
			}
		},
	}
)

func loggerFromFlags(cmd *cli.Command) *slog.Logger {
	var l slog.Level
	if err := l.UnmarshalText([]byte(cmd.String("log"))); err != nil {
		panic(err)
	}
	var h slog.Handler
	switch strings.ToLower(cmd.String("log-format")) {
	case "text":
		h = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l})
	case "json":
		h = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: l})
	}
	return slog.New(h)
}

// metrics configuration
func newMetrics() *metrics.Metrics {
	return &metrics.Metrics{
		TMIMsgsCount: metrics.NewPromCounter(
			prometheus.NewCounter(
				prometheus.CounterOpts{
					Namespace: "robot",
					Subsystem: "tmi",
					Name:      "messages",
					Help:      "Number of PRIVMSGs received from TMI.",
				},
			),
		),
		TMICommandCount: metrics.NewPromCounter(
			prometheus.NewCounter(
				prometheus.CounterOpts{
					Namespace: "robot",
					Subsystem: "tmi",
					Name:      "commands",
					Help:      "Number of command invocations received in Twitch chat.",
				},
			),
		),
		LearnedCount: metrics.NewPromCounter(
			prometheus.NewCounter(
				prometheus.CounterOpts{
					Namespace: "robot",
					Subsystem: "brain",
					Name:      "learned",
					Help:      "Number of messages learned.",
				},
			),
		),
		ForgotCount: metrics.NewPromCounter(
			prometheus.NewCounter(
				prometheus.CounterOpts{
					Namespace: "robot",
					Subsystem: "brain",
					Name:      "forgot",
					Help:      "Number of individual messages deleted. Does not include messages deleted by user or time.",
				},
			),
		),
		SpeakLatency: metrics.NewPromObserverVec(
			prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Buckets:   []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1, 5, 10},
					Namespace: "robot",
					Subsystem: "commands",
					Name:      "speak_latency",
					Help:      "How long it takes for robot to speak once prompted in seconds",
				},
				[]string{"tag", "empty_prompt"},
			),
		),
		LearnLatency: metrics.NewPromObserverVec(
			prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Buckets:   []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1, 5, 10},
					Namespace: "robot",
					Subsystem: "brain",
					Name:      "learn_latency",
					Help:      "How long it takes robot to learn a non discarded message in seconds",
				},
				[]string{"tag"},
			),
		),
		UsedMessagesForGeneration: metrics.NewPromHistogram(
			prometheus.NewHistogram(
				prometheus.HistogramOpts{
					Buckets:   []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
					Namespace: "robot",
					Subsystem: "commands",
					Name:      "used_messages",
					Help:      "How many messages were used while generating a new message",
				},
			),
		),
	}
}

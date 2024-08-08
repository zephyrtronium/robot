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

	"github.com/urfave/cli/v3"

	"github.com/zephyrtronium/robot/brain/sqlbrain"
	"github.com/zephyrtronium/robot/privacy"
)

var app = cli.Command{
	Name:  "robot",
	Usage: "Markov chain chat bot",

	DefaultCommand: "run",
	Commands: []*cli.Command{
		{
			Name:   "init",
			Usage:  "Initialize configured databases",
			Action: cliInit,
			Flags: []cli.Flag{
				&flagConfig,
			},
		},
		{
			Name:   "run",
			Usage:  "Connect to configured chat services",
			Action: cliRun,
			Flags: []cli.Flag{
				&flagConfig,
			},
		},
	},
	Flags: []cli.Flag{
		&flagLog,
		&flagLogFormat,
	},

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

func cliInit(ctx context.Context, cmd *cli.Command) error {
	slog.SetDefault(loggerFromFlags(cmd))
	r, err := os.Open(cmd.String("config"))
	if err != nil {
		return fmt.Errorf("couldn't open config file: %w", err)
	}
	cfg, _, err := Load(ctx, r)
	if err != nil {
		return fmt.Errorf("couldn't load config: %w", err)
	}
	_, sql, priv, err := loadDBs(cfg.DB)
	if err != nil {
		return err
	}
	if sql != nil {
		if err := sqlbrain.Create(ctx, sql); err != nil {
			return fmt.Errorf("couldn't initialize sqlbrain: %w", err)
		}
	}
	if err := privacy.Init(ctx, priv); err != nil {
		return fmt.Errorf("couldn't initialize privacy list: %w", err)
	}
	return nil
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
	robo := New(runtime.GOMAXPROCS(0))
	robo.SetOwner(cfg.Owner.Name, cfg.Owner.Contact)
	if err := robo.SetSecrets(cfg.SecretFile); err != nil {
		return err
	}
	kv, sql, priv, err := loadDBs(cfg.DB)
	if err != nil {
		return err
	}
	if err := robo.SetSources(ctx, kv, sql, priv); err != nil {
		return err
	}
	if md.IsDefined("tmi") {
		if err := robo.SetTMI(ctx, cfg.TMI); err != nil {
			return err
		}
		if err := robo.InitTwitchUsers(ctx, &cfg.TMI.Owner, cfg.Twitch); err != nil {
			return err
		}
		if err := robo.SetTwitchChannels(ctx, cfg.Global, cfg.Twitch); err != nil {
			return err
		}
	}
	return robo.Run(ctx)
}

var (
	flagConfig = cli.StringFlag{
		Name:     "config",
		Required: true,
		Usage:    "TOML config file",
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

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"

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
				&flagOrder,
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
	order := int(cmd.Int("order"))
	r, err := os.Open(cmd.String("config"))
	if err != nil {
		return fmt.Errorf("couldn't open config file: %w", err)
	}
	cfg, _, err := Load(ctx, r)
	if err != nil {
		return fmt.Errorf("couldn't load config: %w", err)
	}
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

func cliRun(ctx context.Context, cmd *cli.Command) error {
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
	brain, priv, err := loadDBs(ctx, cfg.DB)
	if err != nil {
		return err
	}
	if err := robo.SetSources(ctx, brain, priv); err != nil {
		return err
	}
	if md.IsDefined("tmi") {
		if err := robo.SetTMI(ctx, cfg.TMI); err != nil {
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

	flagOrder = cli.IntFlag{
		Name:     "order",
		Required: true,
		Usage:    "Prefix length for Markov chains",
		Action: func(ctx context.Context, cmd *cli.Command, i int64) error {
			if i <= 0 {
				return errors.New("order must be positive")
			}
			return nil
		},
	}
)

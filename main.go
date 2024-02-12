package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/urfave/cli/v3"
)

var app = cli.Command{
	Name:  "robot",
	Usage: "Markov chain chat bot",

	DefaultCommand: "run",
	Commands: []*cli.Command{
		{
			Name:  "init",
			Usage: "Initialize configured databases",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				order := int(cmd.Int("order"))
				cfg, err := os.Open(cmd.String("config"))
				if err != nil {
					return fmt.Errorf("couldn't open config file: %w", err)
				}
				if err := Init(ctx, cfg, order); err != nil {
					return err
				}
				return nil
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
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
				},
				&cli.IntFlag{
					Name:     "order",
					Required: true,
					Usage:    "Prefix length for Markov chains",
					Action: func(ctx context.Context, cmd *cli.Command, i int64) error {
						if i <= 0 {
							return errors.New("order must be positive")
						}
						return nil
					},
				},
			},
		},
		{
			Name:  "run",
			Usage: "Connect to configured chat services",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				cfg, err := os.Open(cmd.String("config"))
				if err != nil {
					return fmt.Errorf("couldn't open config file: %w", err)
				}
				robo, err := Load(ctx, cfg)
				if err != nil {
					return fmt.Errorf("couldn't load config: %w", err)
				}
				return robo.Run(ctx)
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
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
				},
			},
		},
	},

	Authors: []any{
		"Branden J Brown  @zephyrtronium",
	},
	Copyright: "Copyright 2023 Branden J Brown",
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

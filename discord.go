package main

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/zephyrtronium/robot/brain"
	"log/slog"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type MessageReceiver struct {
	session *discordgo.Session
}

func (robo *Robot) NewDiscord(ctx context.Context, token string) (*MessageReceiver, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	receiver := &MessageReceiver{
		session: session,
	}

	// Enable message receive intent
	session.Identify.Intents = discordgo.IntentGuilds | discordgo.IntentGuildBans | discordgo.IntentsGuildMessages

	// Register message handler
	session.AddHandler(func(session *discordgo.Session, event *discordgo.MessageCreate) {
		robo.onDiscordMessage(ctx, session, event)
	})

	// Register slash commands on startup
	session.AddHandler(func(session *discordgo.Session, event *discordgo.Ready) {
		permission := int64(discordgo.PermissionManageMessages)
		commands := []*discordgo.ApplicationCommand{
			{
				Name:        "say",
				Description: "Say something",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "prompt",
						Description: "Prompt to use for generating a message",
						Required:    true,
					},
				},
			},
			{
				Name:                     "forget",
				DefaultMemberPermissions: &permission,
				Description:              "Forget something",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "prompt",
						Description: "Prompt to search for and delete messages",
						Required:    true,
					},
				},
			},
		}
		// Register as global commands
		_, err := session.ApplicationCommandBulkOverwrite(event.Application.ID, "", commands)
		if err != nil {
			_ = fmt.Errorf("failed to update slash commands: %w", err)
		}
	})

	// Forget messages on deletion
	session.AddHandler(func(session *discordgo.Session, event *discordgo.MessageDelete) {
		ch, _ := robo.channels.Load(event.GuildID)
		if ch == nil {
			return
		}
		err := robo.brain.Forget(ctx, ch.Learn, event.Message.ID)
		if err != nil {
			_ = fmt.Errorf("failed to delete message: %w", err)
		}
	})

	// Handle slash commands
	session.AddHandler(func(session *discordgo.Session, event *discordgo.InteractionCreate) {
		if event.Type != discordgo.InteractionApplicationCommand {
			return
		}
		ch, _ := robo.channels.Load(event.GuildID)
		if ch == nil {
			return
		}
		data := event.ApplicationCommandData()
		if data.Name == "say" {
			prompt := data.Options[0].StringValue()
			response := robo.discordThink(ctx, ch.Learn, prompt)
			_ = session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content:         response,
					AllowedMentions: nil,
				},
			})
		} else if data.Name == "forget" {
			prompt := data.Options[0].StringValue()
			n := 64
			messages := make([]brain.Message, n)
			n, _, err := robo.brain.Recall(ctx, ch.Learn, "", messages)
			if err != nil {
				_ = session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags:           discordgo.MessageFlagsEphemeral,
						Content:         err.Error(),
						AllowedMentions: nil,
					},
				})
				return
			}
			deleted := 0
			for _, message := range messages {
				if !strings.Contains(message.Text, prompt) {
					continue
				}
				_ = robo.brain.Forget(ctx, ch.Learn, message.ID)
				deleted++
			}
			_ = session.InteractionRespond(event.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:           discordgo.MessageFlagsEphemeral,
					Content:         "Deleted " + strconv.Itoa(deleted) + " messages from the database.",
					AllowedMentions: nil,
				},
			})
		}
	})

	return receiver, nil
}

func (robo *Robot) onDiscordMessage(ctx context.Context, session *discordgo.Session, event *discordgo.MessageCreate) {
	// Ignore messages sent by bots
	if event.Author.Bot {
		return
	}
	log := slog.With(slog.String("trace", event.Message.ID), slog.String("in", event.GuildID))

	// Find configuration for this server
	ch, _ := robo.channels.Load(event.GuildID)
	if ch == nil {
		return
	}

	// If this is a blocked message we don't want to interact
	if ch.Block.MatchString(event.Content) {
		return
	}

	// Check for the channel being silent. This prevents learning, copypasta,
	// and random speaking (among other things), which happens to be all the
	// rest of this function.
	if s := ch.SilentTime(); event.Timestamp.Before(s) {
		log.DebugContext(ctx, "channel is silent", slog.Time("until", s))
		return
	}

	// TODO: Learn from this user only if not disabled, not sure if applicable to discord
	// TODO: Meme detector stuff, possibly not necessary in discord?
	robo.discordLearn(ctx, ch.Learn, event)

	// Now we can check rate limits
	t := time.Now()
	r := ch.Rate.ReserveN(t, 1)
	if d := r.DelayFrom(t); d > 0 {
		log.InfoContext(ctx, "rate limited",
			slog.String("action", "speak"),
			slog.String("delay", d.String()),
		)
		r.CancelAt(t)
		return
	}

	// Attempt to handle textual commands
	if strings.HasPrefix(event.Content, session.State.User.Mention()) {
		// Message starts by mentioning robot
		trimmed := strings.TrimSpace(strings.TrimPrefix(event.Content, session.State.User.Mention()))
		parser := regexp.MustCompile(`^(?i:say|generate)\s*(?i:something)?\s*(?i:starting)?\s*(?i:with)?\s+(.*)`)
		matches := parser.FindStringSubmatch(trimmed)
		if matches != nil && len(matches) > 1 {
			prompt := matches[1]
			robo.discordSend(ctx, session, event.Message.Reference(), event.ChannelID, ch.Send, prompt)
		} else {
			robo.discordSend(ctx, session, event.Message.Reference(), event.ChannelID, ch.Send, "")
		}
		return
	}

	// Not rate limited and not a command so check for random message chance
	if rand.Float64() > ch.Responses {
		return
	}
	robo.discordSend(ctx, session, nil, event.ChannelID, ch.Send, "")
}

func (robo *Robot) discordSend(ctx context.Context, session *discordgo.Session, reference *discordgo.MessageReference, channelId string, tag string, prompt string) {
	response := robo.discordThink(ctx, tag, prompt)
	if response == "" {
		return
	}
	message := &discordgo.MessageSend{
		Content:         response,
		AllowedMentions: nil,
		Reference:       reference,
	}
	_, err := session.ChannelMessageSendComplex(
		channelId,
		message,
	)
	if err != nil {
		_ = fmt.Errorf("failed to send discord message: %w", err)
	}
}

func (robo *Robot) discordThink(ctx context.Context, tag string, prompt string) string {
	m, _, err := brain.Think(ctx, robo.brain, tag, prompt)
	if err != nil {
		_ = fmt.Errorf("couldn't think: %w", err)
		return ""
	}
	return m
}

func (robo *Robot) discordLearn(ctx context.Context, tag string, event *discordgo.MessageCreate) {
	userHash := robo.hashes().Hash(event.Author.ID, event.GuildID, event.Timestamp)
	message := brain.Message{
		Sender:      userHash,
		ID:          event.Message.ID,
		To:          event.GuildID,
		Text:        event.Content,
		Timestamp:   event.Timestamp.UnixMilli(),
		IsModerator: event.Member.Permissions&discordgo.PermissionManageMessages != 0,
		IsElevated:  false, // TODO: Possibly some other flag?
	}
	err := brain.Learn(ctx, robo.brain, tag, &message)
	if err != nil {
		_ = fmt.Errorf("failed to learn message: %w", err)
	}
}

// Start opens the Discord websocket connection.
func (mr *MessageReceiver) Start() error {
	return mr.session.Open()
}

// Close closes the Discord websocket connection.
func (mr *MessageReceiver) Close() error {
	return mr.session.Close()
}

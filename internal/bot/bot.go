// Package bot wires a Discord session to a Store and a putt.day Collector:
// it registers slash commands and reacts to messages containing share links.
package bot

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/alexbathome/albatross/pkg/store"
)

// Bot wires a Discord session to a Store and a putt.day Collector.
type Bot struct {
	session        *discordgo.Session
	store          store.Store
	collector      Collector
	commandGuildID string
	registered     []*discordgo.ApplicationCommand
}

// New constructs a Bot. It does not connect to Discord until Open is called.
func New(token string, st store.Store, collector Collector, commandGuildID string) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("creating discord session: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent

	b := &Bot{session: session, store: st, collector: collector, commandGuildID: commandGuildID}
	session.AddHandler(b.handleMessageCreate)
	session.AddHandler(b.handleInteractionCreate)
	return b, nil
}

// Open connects to Discord and registers slash commands.
func (b *Bot) Open(ctx context.Context) error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("opening discord session: %w", err)
	}
	for _, def := range commandDefs {
		cmd, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, b.commandGuildID, def)
		if err != nil {
			return fmt.Errorf("registering command %s: %w", def.Name, err)
		}
		b.registered = append(b.registered, cmd)
	}
	return nil
}

// Close disconnects the Discord session.
func (b *Bot) Close() error {
	return b.session.Close()
}

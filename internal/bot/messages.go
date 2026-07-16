package bot

import (
	"context"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Author can be nil for some non-user messages (e.g. certain system or
	// webhook events); there's no score to attribute in that case.
	if m.Author == nil {
		return
	}
	msg := IncomingMessage{
		Content:   m.Content,
		IsBot:     m.Author.Bot,
		AuthorID:  m.Author.ID,
		GuildID:   m.GuildID,
		ChannelID: m.ChannelID,
		MessageID: m.ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	emoji := ProcessMessage(ctx, b.collector, b.store, msg, b.commandGuildID)
	if emoji == "" {
		return
	}
	if err := s.MessageReactionAdd(m.ChannelID, m.ID, emoji); err != nil {
		log.Printf("adding reaction to message %s: %v", m.ID, err)
	}
}

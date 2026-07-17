package bot

import (
	"context"
	"errors"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/alexbathome/albatross/pkg/store"
)

const (
	defaultTopCount = 10
	maxTopCount     = 25 // guards against Discord's 2000-character message content limit
)

// removeAnyDefaultPermission is the out-of-the-box access level for
// /remove-any: server Administrators only. A guild admin can further
// customize who this applies to (e.g. a "Score Admin" role instead) in
// Server Settings > Integrations > Albatross, entirely on Discord's side —
// no bot config or redeploy required.
var removeAnyDefaultPermission = int64(discordgo.PermissionAdministrator)

var commandDefs = []*discordgo.ApplicationCommand{
	{
		Name:        "score",
		Description: "Show a user's score(s) on a given hole",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionInteger, Name: "hole", Description: "Hole number", Required: true},
			{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "Discord user", Required: true},
		},
	},
	{
		Name:        "top",
		Description: "Leaderboard for a hole, lowest strokes first",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionInteger, Name: "hole", Description: "Hole number", Required: true},
			{Type: discordgo.ApplicationCommandOptionInteger, Name: "count", Description: "Number of entries (default 10)", Required: false},
		},
	},
	{
		Name:        "remove",
		Description: "Remove one of your own recorded scores",
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "link", Description: "The putt.day share link to remove", Required: true},
		},
	},
	{
		Name:                     "remove-any",
		Description:              "Remove any recorded score, regardless of who played it",
		DefaultMemberPermissions: &removeAnyDefaultPermission,
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "link", Description: "The putt.day share link to remove", Required: true},
		},
	},
}

func (b *Bot) handleInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data := i.ApplicationCommandData()
	switch data.Name {
	case "score":
		b.handleScoreCommand(ctx, s, i, data)
	case "top":
		b.handleTopCommand(ctx, s, i, data)
	case "remove":
		b.handleRemoveCommand(ctx, s, i, data)
	case "remove-any":
		b.handleRemoveAnyCommand(ctx, s, i, data)
	}
}

// interactionUserID returns the ID of the user who invoked i: Member is set
// when invoked in a guild, User is set when invoked in a DM.
func interactionUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func (b *Bot) handleScoreCommand(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	hole := int(data.GetOption("hole").IntValue())
	target := data.GetOption("user").UserValue(s)

	scores, err := b.store.UserScores(ctx, hole, target.ID, 25)
	if err != nil {
		respond(s, i, "Something went wrong looking up that score.")
		return
	}
	respond(s, i, FormatUserScores(hole, target.ID, scores))
}

func (b *Bot) handleTopCommand(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	hole := int(data.GetOption("hole").IntValue())
	count := defaultTopCount
	if opt := data.GetOption("count"); opt != nil {
		count = int(opt.IntValue())
	}
	if count > maxTopCount {
		count = maxTopCount
	}
	if count < 1 {
		count = 1
	}

	scores, err := b.store.TopScores(ctx, hole, count)
	if err != nil {
		respond(s, i, "Something went wrong building the leaderboard.")
		return
	}
	respond(s, i, FormatTopScores(hole, count, scores))
}

func (b *Bot) handleRemoveCommand(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	link := data.GetOption("link").StringValue()
	userID := interactionUserID(i)

	removed, err := b.store.DeleteByShareLink(ctx, link, userID, false)
	respondToRemoval(s, i, removed, err)
}

// handleRemoveAnyCommand deletes a share link regardless of who owns it.
// Access to /remove-any is controlled entirely by Discord's own per-command
// permissions (see removeAnyDefaultPermission), not by any check here.
func (b *Bot) handleRemoveAnyCommand(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	link := data.GetOption("link").StringValue()
	userID := interactionUserID(i)

	removed, err := b.store.DeleteByShareLink(ctx, link, userID, true)
	respondToRemoval(s, i, removed, err)
}

func respondToRemoval(s *discordgo.Session, i *discordgo.InteractionCreate, removed bool, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		respond(s, i, "No recorded score found for that share link.")
	case errors.Is(err, store.ErrForbidden):
		respond(s, i, "You can only remove your own scores.")
	case err != nil:
		respond(s, i, "Something went wrong removing that score.")
	case removed:
		respond(s, i, "Removed.")
	default:
		respond(s, i, "Removed. That result had already been superseded by a more recent play with the same score, so your current standing is unchanged.")
	}
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:         content,
			AllowedMentions: &discordgo.MessageAllowedMentions{},
			Flags:           discordgo.MessageFlagsSuppressEmbeds,
		},
	})
}

// Package config loads the bot's runtime configuration from environment
// variables.
package config

import (
	"fmt"
	"os"
)

// Config holds the bot's runtime configuration, loaded from environment variables.
type Config struct {
	// DiscordToken is the bot token used to authenticate with Discord.
	DiscordToken string
	// DBPath is the filesystem path to the bbolt database file.
	DBPath string
	// CommandGuildID is the Discord server (guild) ID the bot serves: slash
	// commands are registered there, and share links posted in any other
	// guild (or in DMs) are ignored. If empty, commands register globally
	// and messages from every guild are processed.
	CommandGuildID string
}

const defaultDBPath = "albatross.db"

// Load reads Config from environment variables.
func Load() (Config, error) {
	token := os.Getenv("ALBATROSS_DISCORD_TOKEN")
	if token == "" {
		return Config{}, fmt.Errorf("ALBATROSS_DISCORD_TOKEN is required")
	}

	dbPath := os.Getenv("ALBATROSS_DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	return Config{
		DiscordToken:   token,
		DBPath:         dbPath,
		CommandGuildID: os.Getenv("ALBATROSS_COMMAND_GUILD_ID"),
	}, nil
}

package config

import "testing"

func TestLoad_RequiresToken(t *testing.T) {
	t.Setenv("ALBATROSS_DISCORD_TOKEN", "")
	t.Setenv("ALBATROSS_DB_PATH", "")
	t.Setenv("ALBATROSS_COMMAND_GUILD_ID", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error when ALBATROSS_DISCORD_TOKEN is unset")
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("ALBATROSS_DISCORD_TOKEN", "test-token")
	t.Setenv("ALBATROSS_DB_PATH", "")
	t.Setenv("ALBATROSS_COMMAND_GUILD_ID", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DiscordToken != "test-token" {
		t.Errorf("DiscordToken = %q, want %q", cfg.DiscordToken, "test-token")
	}
	if cfg.DBPath != defaultDBPath {
		t.Errorf("DBPath = %q, want default %q", cfg.DBPath, defaultDBPath)
	}
	if cfg.CommandGuildID != "" {
		t.Errorf("CommandGuildID = %q, want empty", cfg.CommandGuildID)
	}
}

func TestLoad_Overrides(t *testing.T) {
	t.Setenv("ALBATROSS_DISCORD_TOKEN", "test-token")
	t.Setenv("ALBATROSS_DB_PATH", "/tmp/custom.db")
	t.Setenv("ALBATROSS_COMMAND_GUILD_ID", "12345")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DBPath != "/tmp/custom.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/tmp/custom.db")
	}
	if cfg.CommandGuildID != "12345" {
		t.Errorf("CommandGuildID = %q, want %q", cfg.CommandGuildID, "12345")
	}
}

// Package albatross wires together config, storage, and the Discord bot,
// and runs them until the process is asked to stop.
package albatross

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexbathome/albatross/internal/bot"
	"github.com/alexbathome/albatross/pkg/puttday"
	"github.com/alexbathome/albatross/pkg/store"
)

// Main loads config, opens the store, starts the bot, and blocks until ctx
// is canceled or a termination signal arrives. args is currently unused.
func Main(ctx context.Context, args []string) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	var (
		fs = flag.NewFlagSet("albatross", flag.ExitOnError)
		discordToken   string
		dbPath         string
		commandGuildId string
	)
	fs.StringVar(&discordToken, "discord-token", "", "the discord bot token (or set ALBATROSS_DISCORD_TOKEN)")
	fs.StringVar(&dbPath, "db-path", os.Getenv("ALBATROSS_DB_PATH"), "the path to the duckdb file")
	fs.StringVar(&commandGuildId, "guild-id", os.Getenv("ALBATROSS_COMMAND_GUILD_ID"), "the discord server/guild id")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if discordToken == "" {
		// don't put sensitive tokens in the fs.StringVar default as it can 
		// leak to stdout when the user runs `-h` or `--help`
		discordToken = os.Getenv("ALBATROSS_DISCORD_TOKEN")
	}

	if discordToken == "" || dbPath == "" {
		return fmt.Errorf("discord-token and db-path are required (as a flag or ALBATROSS_* env var)")
	}

	db, err := store.OpenDuckDb(dbPath)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("closing store: %v", err)
		}
	}()

	collector := puttday.NewCollector()

	b, err := bot.New(discordToken, db, collector, commandGuildId)
	if err != nil {
		return fmt.Errorf("creating bot: %w", err)
	}

	if err := b.Open(ctx); err != nil {
		return fmt.Errorf("opening bot: %w", err)
	}
	defer func() {
		if err := b.Close(); err != nil {
			log.Printf("closing bot: %v", err)
		}
	}()

	log.Println("albatross is running, press ctrl+c to stop")
	<-ctx.Done()
	log.Println("shutting down")
	return nil
}

// Package albatross wires together config, storage, and the Discord bot,
// and runs them until the process is asked to stop.
package albatross

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/alexbathome/albatross/internal/api"
	"github.com/alexbathome/albatross/internal/bot"
	"github.com/alexbathome/albatross/internal/server"
	"github.com/alexbathome/albatross/pkg/puttday"
	"github.com/alexbathome/albatross/pkg/store"
)

// apiShutdownTimeout bounds how long the HTTP API waits for in-flight
// requests to finish during shutdown.
const apiShutdownTimeout = 5 * time.Second

// Main loads config, opens the store, starts the bot, and blocks until ctx
// is canceled or a termination signal arrives. args is currently unused.
func Main(ctx context.Context, args []string) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	var (
		fs             = NewFlagSet("albatross")
		discordToken   = Flag(fs, fs.String, "discord-token", "", "the discord bot token (or set ALBATROSS_DISCORD_TOKEN)", "ALBATROSS_DISCORD_TOKEN", true)
		dbPath         = Flag(fs, fs.String, "db-path", "/tmp/albatross.db", "the path to the duckdb file", "ALBATROSS_DB_PATH", true)
		commandGuildId = Flag(fs, fs.String, "guild-id", "", "the discord server/guild id", "ALBATROSS_COMMAND_GUILD_ID", false)
		apiAddr        = Flag(fs, fs.String, "api-addr", ":8080", "the address the HTTP API listens on", "ALBATROSS_API_ADDR", false)
	)
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	db, err := store.OpenDuckDb(dbPath())
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("closing store: %v", err)
		}
	}()

	collector := puttday.NewCollector()

	b, err := bot.New(discordToken(), db, collector, commandGuildId())
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

	srv := server.NewServer(db)
	api.NewAPI(srv).RegisterRoutes()

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		log.Printf("api listening on %s", apiAddr())
		if err := srv.ListenAndServe(apiAddr()); err != nil {
			return fmt.Errorf("api server: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		<-gctx.Done()
		log.Println("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), apiShutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	})

	log.Println("albatross is running, press ctrl+c to stop")
	return g.Wait()
}

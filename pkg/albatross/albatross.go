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
		fs             = flag.NewFlagSet("albatross", flag.ExitOnError)
		discordToken   string
		dbPath         string
		commandGuildId string
		apiAddr        string
	)
	fs.StringVar(&discordToken, "discord-token", "", "the discord bot token (or set ALBATROSS_DISCORD_TOKEN)")
	fs.StringVar(&dbPath, "db-path", os.Getenv("ALBATROSS_DB_PATH"), "the path to the duckdb file")
	fs.StringVar(&commandGuildId, "guild-id", os.Getenv("ALBATROSS_COMMAND_GUILD_ID"), "the discord server/guild id")
	fs.StringVar(&apiAddr, "api-addr", envOr("ALBATROSS_API_ADDR", ":8080"), "the address the HTTP API listens on")
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

	srv := server.NewServer(db)
	api.NewAPI(srv).RegisterRoutes()

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		log.Printf("api listening on %s", apiAddr)
		if err := srv.ListenAndServe(apiAddr); err != nil {
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

// envOr returns the value of the environment variable key, or def if unset.
func envOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

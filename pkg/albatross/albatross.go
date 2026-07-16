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

	"github.com/alexbathome/albatross/internal/bot"
	"github.com/alexbathome/albatross/internal/config"
	"github.com/alexbathome/albatross/pkg/puttday"
	"github.com/alexbathome/albatross/pkg/store"
)

// Main loads config, opens the store, starts the bot, and blocks until ctx
// is canceled or a termination signal arrives. args is currently unused.
func Main(ctx context.Context, args []string) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer func() {
		if err := st.Close(); err != nil {
			log.Printf("closing store: %v", err)
		}
	}()

	collector := puttday.NewCollector()

	b, err := bot.New(cfg.DiscordToken, st, collector, cfg.CommandGuildID)
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

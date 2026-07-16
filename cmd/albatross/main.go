// Command albatross runs the Discord bot.
package main

import (
	"context"
	"log"
	"os"

	"github.com/alexbathome/albatross/pkg/albatross"
)

func main() {
	if err := albatross.Main(context.Background(), os.Args); err != nil {
		log.Fatalf("error: %v", err)
	}
}

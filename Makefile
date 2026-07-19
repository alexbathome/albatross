# Build order matters: web/dist is embedded into the Go binary at compile
# time (web/embed.go), so the frontend must be built before the server.

BINARY := bin/albatross

.PHONY: all build web generate test vet lint check clean docker

all: build

# Build the frontend into web/dist. Uses a plain vite build because the
# canonical `deno task build` (vue-tsc + vite) is blocked on vue-tsc
# resolving .vue imports under Deno's node_modules layout — see web/README.md.
# vite empties dist/ first, so restore the .gitkeep that go:embed relies on.
web:
	cd web && deno install && deno run -A npm:vite build && touch dist/.gitkeep

# Build the server binary with the frontend embedded. CGO is required by
# the DuckDB driver.
build: web
	CGO_ENABLED=1 go build -o $(BINARY) ./cmd/albatross

# Regenerate the swagger docs (docs/) from the handler annotations.
generate:
	go generate ./...

test:
	go test ./...

vet:
	go vet ./...

# The repo pins golangci-lint v2 as a go tool; a globally installed v1
# rejects the v2 config, so always run it through `go tool`.
lint:
	go tool golangci-lint run

# Everything CI would care about.
check: vet lint test

# Removes build output, keeping web/dist/.gitkeep — go:embed needs web/dist
# to hold at least one file for `go build` to succeed without a frontend build.
clean:
	rm -rf bin web/dist
	mkdir -p web/dist && touch web/dist/.gitkeep

docker:
	docker build -t albatross .

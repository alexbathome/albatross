# Hacking on albatross

## Building and testing

The Makefile drives everything in the right order (frontend first — its build output is embedded into the Go binary):

| Target | What it does |
|---|---|
| `make` / `make build` | Build the frontend, then the server binary to `bin/albatross` with the frontend embedded (CGO required by DuckDB) |
| `make web` | Build just the frontend into `web/dist/` |
| `make generate` | Regenerate the swagger docs in `docs/` from the handler annotations (`go generate ./...`) |
| `make test` | `go test ./...` |
| `make check` | Everything CI would care about: vet + lint + test |
| `make lint` | golangci-lint, via `go tool` (the repo pins v2 as a Go tool; a globally installed v1 rejects the config) |
| `make clean` | Remove `bin/` and the frontend build (restores `web/dist/.gitkeep`, which `go:embed` needs to compile without a frontend build) |
| `make docker` | Build the deployment image |

Prerequisites: Go, [Deno](https://deno.com/) (frontend), and Docker for `make docker`.

## Deploying (Fly.io)

The app is already configured via `fly.toml` (app `albatross`, region `iad`, builds from the repo `Dockerfile` into a distroless static binary, env `ALBATROSS_DB_PATH=/data/albatross.db`, volume `albatross_data` mounted at `/data`, shared-cpu-1x/256mb VM).

First deploy:

1. `fly apps create albatross` (or `fly launch` — decline if it offers to overwrite the existing `fly.toml`/`Dockerfile`)
2. `fly volumes create albatross_data --region iad --size 1` (region must match `primary_region` in `fly.toml`; change both together if deploying elsewhere)
3. `fly secrets set ALBATROSS_DISCORD_TOKEN=<your bot token>` — never put this in `fly.toml` or any committed file
4. Fill in `ALBATROSS_COMMAND_GUILD_ID` in `fly.toml` with the target server's guild ID. This both registers the slash commands on that server and makes the bot ignore share links posted anywhere else — if it's ever added to another server, messages there are dropped rather than recorded into the shared database.
5. `fly deploy`

The `albatross_data` volume persists across redeploys and is reattached automatically, so the DuckDB database survives updates.

## Web frontend

The web UI lives in `web/` (Vue 3 + TypeScript + Vite, managed with [Deno](https://deno.com/) — see `web/README.md` for the layout and conventions).

Local development (two processes, run from the repo root):

1. Start the Go app so the API is on `:8080`.
2. `cd web && deno task dev` — Vite dev server on http://localhost:5173 with hot reload, proxying `/api` to `:8080` (see `web/vite.config.ts`).

Production build: `make build` (or `make web` for the frontend alone). The frontend build lands in `web/dist/`, which `web/embed.go` embeds into the Go binary via `go:embed` — the server serves it at `/` (with an `index.html` fallback for client-side routes like `/holes/66`, wired in `internal/server/frontend.go`). Two things to know:

- **Rebuild the Go binary after rebuilding the frontend** — the embed happens at Go compile time, not at runtime. `make build` always does both in order.
- The canonical build script `deno task build` (vue-tsc type-check + vite) currently fails: vue-tsc can't resolve `.vue` imports under Deno's node_modules layout. Until that's fixed, `make web` (and the Dockerfile) use a plain vite build without the type-check.

If the frontend has never been built, `go build` still works — `web/dist/.gitkeep` is committed to keep the embed target present — and the server just serves the API with a 404 at `/`.

`fly deploy` needs no extra steps: the Dockerfile builds the frontend in a Deno stage and copies `dist/` in before compiling the Go binary. Goreleaser has no such hook, so its binaries embed whatever `web/dist/` contains on the release machine — build the frontend first if releasing that way.

## Adding Albatross to a server

1. Create an application at the [Discord Developer Portal](https://discord.com/developers/applications).
2. In the **Bot** tab, enable **Message Content Intent** under Privileged Gateway Intents — required, or the bot receives empty message content and can never detect pasted putt.day share links.
3. In the **Bot** tab, get/reset the bot token — this is `ALBATROSS_DISCORD_TOKEN`. Treat it as a secret.
4. Enable Developer Mode (User Settings → Advanced → Developer Mode), then right-click the target server's icon → Copy Server ID — this is `ALBATROSS_COMMAND_GUILD_ID`.
5. Build an invite URL via OAuth2 → URL Generator with scopes `bot` and `applications.commands`, and bot permissions View Channel, Read Message History, Add Reactions (for the ⛳/⚠️ reactions on detected share links), optionally Send Messages. Permission integer `68672` covers that set:

   ```
   https://discord.com/api/oauth2/authorize?client_id=YOUR_APP_ID&permissions=68672&scope=bot%20applications.commands
   ```

   Swap in the real application/client ID from the General Information tab, then open the URL and add the bot to the server.

## Managing `/remove-any` access

`/remove` is self-service — a user can only remove their own recorded scores. `/remove-any` bypasses ownership and can remove any recorded score; out of the box it's restricted to server Administrators via the command's default permissions at registration.

A server admin can change who's allowed to use it (e.g. grant it to a "Score Admin" role instead) from Discord's UI: **Server Settings → Integrations → Albatross**, find `/remove-any`, and customize its allowed roles/members. No bot config change or redeploy needed, and it can be changed anytime.

# Hacking on albatross

## Deploying (Fly.io)

The app is already configured via `fly.toml` (app `albatross`, region `iad`, builds from the repo `Dockerfile` into a distroless static binary, env `ALBATROSS_DB_PATH=/data/albatross.db`, volume `albatross_data` mounted at `/data`, shared-cpu-1x/256mb VM).

First deploy:

1. `fly apps create albatross` (or `fly launch` — decline if it offers to overwrite the existing `fly.toml`/`Dockerfile`)
2. `fly volumes create albatross_data --region iad --size 1` (region must match `primary_region` in `fly.toml`; change both together if deploying elsewhere)
3. `fly secrets set ALBATROSS_DISCORD_TOKEN=<your bot token>` — never put this in `fly.toml` or any committed file
4. Fill in `ALBATROSS_COMMAND_GUILD_ID` in `fly.toml` with the target server's guild ID. This both registers the slash commands on that server and makes the bot ignore share links posted anywhere else — if it's ever added to another server, messages there are dropped rather than recorded into the shared database.
5. `fly deploy`

The `albatross_data` volume persists across redeploys and is reattached automatically, so the bbolt database survives updates.

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

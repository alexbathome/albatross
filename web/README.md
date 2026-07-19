# Albatross web UI

Vue 3 + TypeScript + Vite frontend for the albatross REST API, managed with
[Deno](https://deno.com/).

## Commands

Run from this directory:

- `deno task dev` — dev server on http://localhost:5173, proxying `/api` to the
  Go API on `localhost:8080` (see `vite.config.ts`)
- `deno task build` — type-check and build to `dist/`, which the Go server
  embeds (`web/embed.go`) and serves at `/`; rebuild the Go binary after
  rebuilding the frontend to pick up changes
- `deno task preview` — serve the production build locally

Known issue: the `vue-tsc` type-check step in `deno task build` currently fails
to resolve `.vue` imports under Deno's node_modules layout. Until that's fixed,
`deno run -A npm:vite build` builds without the type-check.

## Layout

```
src/
  api/          typed fetch client for the REST API (client.ts, types.ts)
  components/   presentational components (tables, search)
  composables/  shared reactive helpers (useAsync)
  views/        routed pages: HomeView (/), HoleView (/holes/:hole)
  router/       route table
  style.css     design tokens (light/dark via prefers-color-scheme) and
                shared card/table/pill classes
```

Views own data fetching and pass results down; components are presentational.
The one exception is `UserSearch.vue`, which owns its own debounced search.
Dark mode follows the system setting — every color in a component must come
from the tokens in `style.css`, never a raw hex value.

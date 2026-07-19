# syntax=docker/dockerfile:1

FROM denoland/deno:2.9.3 AS webbuild
WORKDIR /src/web
COPY web/package.json web/deno.lock ./
RUN deno install
COPY web/ .
# deno task build (vue-tsc + vite) is blocked on vue-tsc resolving .vue files
# under Deno's node_modules layout — see web/README.md. Build without the
# type-check until that's fixed.
RUN deno run -A npm:vite build

FROM golang:1.26-bookworm AS build
RUN apt-get update && apt-get install -y --no-install-recommends build-essential \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=webbuild /src/web/dist ./web/dist
RUN CGO_ENABLED=1 go build -o /out/albatross ./cmd/albatross

FROM gcr.io/distroless/cc-debian12
COPY --from=build /out/albatross /albatross
ENTRYPOINT ["/albatross"]

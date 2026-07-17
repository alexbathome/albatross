# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS build
RUN apt-get update && apt-get install -y --no-install-recommends build-essential \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /out/albatross ./cmd/albatross

FROM gcr.io/distroless/cc-debian12
COPY --from=build /out/albatross /albatross
ENTRYPOINT ["/albatross"]

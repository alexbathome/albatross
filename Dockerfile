# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/albatross ./cmd/albatross

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/albatross /albatross
ENTRYPOINT ["/albatross"]

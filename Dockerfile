FROM golang:alpine AS build

RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=1 go build -o /app/server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=build /app/server /app/server

ENV HTTP_ADDR=:8080
ENV DB_METADATA_PATH=/data/spotify_metadata.sqlite3
ENV DB_PLAYLISTS_PATH=/data/spotify_playlists.sqlite3
ENV DB_TRACK_URLS_PATH=/data/spotify_track_urls.sqlite3

EXPOSE 8080
ENTRYPOINT ["/app/server"]

# syntax=docker/dockerfile:1.7
FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/arnav.blog .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata wget su-exec \
    && addgroup -S blog \
    && adduser -S -G blog blog

WORKDIR /app
COPY --from=build /out/arnav.blog /usr/local/bin/arnav.blog
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod 0755 /usr/local/bin/docker-entrypoint.sh \
    && mkdir -p /app/data/media \
    && chown -R blog:blog /app

ENV PORT=8080 \
    MEDIA_DIR=/app/data/media

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]

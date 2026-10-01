FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/uptime-monitor ./cmd/app

FROM alpine:3.21

RUN addgroup -S app && adduser -S -D -H -G app app \
    && apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /out/uptime-monitor /app/uptime-monitor
COPY web /app/web

USER app
EXPOSE 8080
ENTRYPOINT ["/app/uptime-monitor"]

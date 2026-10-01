FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/uptime-monitor ./cmd/app

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
	&& addgroup -S app \
	&& adduser -S -D -H -G app app

WORKDIR /app
COPY --from=builder /out/uptime-monitor .
COPY web ./web

USER app
ENTRYPOINT ["./uptime-monitor"]

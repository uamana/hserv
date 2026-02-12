# Stage 1: build
FROM golang:1.25-alpine AS builder

WORKDIR /build

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/hserv ./cmd/hserv

# Stage 2: minimal runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -g "" appuser
WORKDIR /app

COPY --from=builder /build/hserv .

USER appuser

EXPOSE 6443

# All params configurable via env (defaults match app flags):
#   HSERV_ADDR, HSERV_ROOT, HSERV_SID, HSERV_EXT, HSERV_MIME,
#   HSERV_BSIZE, HSERV_CERT, HSERV_KEY
ENTRYPOINT ["/bin/sh", "-c", "exec /app/hserv \
  -addr \"${HSERV_ADDR:-:6443}\" \
  -root \"${HSERV_ROOT:-.}\" \
  -sid \"${HSERV_SID:-sid}\" \
  -uid \"${HSERV_UID:-uid}\" \
  -ext \"${HSERV_EXT:-.ts}\" \
  -mime \"${HSERV_MIME:-video/mp2t}\" \
  -bsize \"${HSERV_BSIZE:-1024}\" \
  -cert \"${HSERV_CERT:-}\" \
  -key \"${HSERV_KEY:-}\""]

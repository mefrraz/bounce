# Stage 1: build
FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X github.com/mefrraz/bounce/internal/api.Version=$(cat VERSION)" -o /bounce ./cmd/server

# Stage 2: runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget
COPY --from=builder /bounce /usr/local/bin/bounce

ENV BOUNCE_PORT=3001
EXPOSE 3001

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:3001/health || exit 1

ENTRYPOINT ["/usr/local/bin/bounce"]

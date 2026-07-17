# Stage 1: build
FROM golang:alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN cp VERSION internal/api/VERSION
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bounce ./cmd/server

# Stage 2: runtime (~15MB)
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata wget
COPY --from=builder /bounce /usr/local/bin/bounce
EXPOSE 3001 80 443
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:3001/health || exit 1
ENTRYPOINT ["/usr/local/bin/bounce"]

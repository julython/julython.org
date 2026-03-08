# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/july ./cmd/server

# Production stage
FROM alpine:3.20 AS prod

WORKDIR /app

# Install ca-certificates for HTTPS and tzdata for timezones
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/bin/july /app/bin/july
COPY migrations /app/migrations

EXPOSE 8000

ENTRYPOINT ["/app/bin/july"]
CMD ["serve"]
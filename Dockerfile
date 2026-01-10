ARG GO_VERSION='1.21'

# Build stage
FROM golang:${GO_VERSION}-alpine AS build

# Install build dependencies (gcc required for go-sqlite3)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go ./

# Build the application
# CGO_ENABLED=1 is required for go-sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o sqlite-s3-backup .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests to S3
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the binary from build stage
COPY --from=build /app/sqlite-s3-backup .

# Create a directory for SQLite databases (optional)
RUN mkdir -p /data

# Run the application
CMD ["./sqlite-s3-backup"]

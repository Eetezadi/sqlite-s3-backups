ARG GO_VERSION='1.21'

# Build stage
FROM golang:${GO_VERSION}-alpine AS build

# Install build dependencies (gcc required for libSQL)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the application
# CGO_ENABLED=1 is required for libSQL
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o libsql-s3-backup .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests to S3 and libSQL
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the binary from build stage
COPY --from=build /app/libsql-s3-backup .

# Run the application
CMD ["./libsql-s3-backup"]

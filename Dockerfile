# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o commander cmd/server/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

# Install common CLI tools (optional - add the ones you need)
RUN apk --no-cache add \
    curl \
    wget \
    ffmpeg \
    rsync

# Note: For yt-dlp and gallery-dl, you'll need Python
# RUN apk --no-cache add python3 py3-pip
# RUN pip3 install yt-dlp gallery-dl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/commander .

# Copy static files
COPY --from=builder /app/web ./web

# Copy default config
COPY --from=builder /app/config ./config

# Expose port
EXPOSE 8080

# Run the application
CMD ["./commander"]

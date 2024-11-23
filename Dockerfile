# Base image for Go
FROM golang:1.20 AS builder

# Set environment variables
ENV CGO_ENABLED=1 \
    GOOS=linux \
    GOARCH=amd64

# Set working directory inside the container
WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o main .

# Final stage for running the app
FROM debian:bullseye-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy the database file and assets
COPY --from=builder /app/examplestore.db ./examplestore.db
COPY --from=builder /app/assets ./assets
COPY --from=builder /app/templates ./templates

# Copy the built binary from the builder stage
COPY --from=builder /app/main .

# Expose the port your Go application listens on
EXPOSE 8080

# Command to run the application
CMD ["./main"]

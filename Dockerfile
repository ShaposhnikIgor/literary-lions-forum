# Stage 1: Build the application
FROM golang:1.20 AS builder
WORKDIR /app

# Install necessary dependencies for CGO (SQLite requires gcc and libsqlite3-dev)
RUN apt-get update && apt-get install -y gcc libsqlite3-dev

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Enable CGO and set Go build flags
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o literary-jo main.go

# Stage 2: Final runtime environment
FROM ubuntu:latest
WORKDIR /app

# Install certificates and required libraries for compatibility
RUN apt-get update && apt-get install -y ca-certificates libsqlite3-0 && rm -rf /var/lib/apt/lists/*

# Copy the compiled binary and necessary assets from the builder stage
COPY --from=builder /app/literary-jo .
COPY --from=builder /app/assets ./assets
COPY --from=builder /app/internal/db/forum.db ./internal/db/forum.db

EXPOSE 8080
CMD ["./literary-jo"]

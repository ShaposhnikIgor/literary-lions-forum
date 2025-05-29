# Define Go version and base image as arguments for flexibility
ARG GO_VERSION
ARG BASE_IMAGE

# **Stage 1: Build Stage**
# Use the official Go image with the specified version for building the application
FROM golang:${GO_VERSION} AS builder

# Set the working directory for the build process
WORKDIR /app

# Install necessary dependencies for CGO since SQLite requires GCC
RUN apt-get update && apt-get install -y gcc libsqlite3-dev

# Copy module definition files to download dependencies
COPY go.mod go.sum ./

# Download the Go modules specified in go.mod and go.sum
RUN go mod download

# Copy the entire source code into the working directory
COPY . .

# Compile the Go application with CGO enabled for SQLite support
# Target OS: Linux; Architecture: AMD64; Output: 'literary-jo'
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o literary-jo main.go

# **Stage 2: Runtime Environment**
# Use the base image specified in the build arguments for a minimal runtime
FROM ${BASE_IMAGE}

# Set the working directory inside the runtime container
WORKDIR /app

# Install necessary runtime libraries and certificates for the application
# Handle dependencies differently based on the type of base image:
# - Alpine: Add CA certificates and SQLite libraries using `apk`
# - Debian/Ubuntu: Add CA certificates and SQLite libraries using `apt-get`
RUN if [ "${BASE_IMAGE}" = "alpine" ]; then \
      apk --no-cache add ca-certificates sqlite-libs; \
    elif [ "${BASE_IMAGE}" = "debian" ] || [ "${BASE_IMAGE}" = "ubuntu" ]; then \
      apt-get update && apt-get install -y ca-certificates libsqlite3-0 && rm -rf /var/lib/apt/lists/*; \
    fi

# Copy the compiled binary from the builder stage into the runtime environment
COPY --from=builder /app/literary-jo .

# Copy the assets directory from the builder stage (e.g., static files for the application)
COPY --from=builder /app/assets ./assets

# Copy the SQLite database file into the runtime container
COPY --from=builder /app/internal/db/forum.db ./internal/db/forum.db

# Expose the application on port 8080 to allow access
EXPOSE 8080

# Define the default command to run the compiled Go application
CMD ["./literary-jo"]

#!/bin/bash
# Specifies that this script should be executed with the Bash shell.

# Get the current version of Go installed on the system.
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
# `go version` retrieves the installed Go version.
# `awk '{print $3}'` extracts the third word in the output, which is the version string (e.g., "go1.20.3").
# `sed 's/go//'` removes the "go" prefix, leaving only the version number (e.g., "1.20.3").

# Check if Go is installed by verifying if GO_VERSION is empty.
if [ -z "$GO_VERSION" ]; then
  echo "Go is not installed on this system."
  # Prints a message to the user indicating Go is not installed.
  exit 1
  # Exits the script with a non-zero status to indicate an error.
fi

# Specify the base Docker image, defaulting to `ubuntu:latest` if no argument is provided.
BASE_IMAGE=${1:-ubuntu:latest}
# Uses the first argument passed to the script (`$1`) or sets a default value (`ubuntu:latest`) if no argument is provided.

echo "Building Docker image with Go version $GO_VERSION and base image $BASE_IMAGE..."
# Prints a message to inform the user about the Docker image being built with specific parameters.

# Build the Docker image with specified Go version and base image.
docker build --build-arg GO_VERSION=$GO_VERSION --build-arg BASE_IMAGE=$BASE_IMAGE -t literary-jo:latest .
# `--build-arg GO_VERSION=$GO_VERSION`: Passes the Go version as a build argument to the Dockerfile.
# `--build-arg BASE_IMAGE=$BASE_IMAGE`: Passes the base image as a build argument to the Dockerfile.
# `-t literary-jo:latest`: Tags the resulting Docker image as `literary-jo:latest`.

# Remove any existing container named `literary-jo-container` if it exists.
docker rm -f literary-jo-container 2>/dev/null
# `docker rm -f`: Forcefully stops and removes a container.
# `2>/dev/null`: Suppresses any error messages if the container does not exist.

# Run a new container using the built Docker image.
docker run --name literary-jo-container -d -p 8080:8080 literary-jo:latest
# `--name literary-jo-container`: Names the container `literary-jo-container`.
# `-d`: Runs the container in detached mode (in the background).
# `-p 8080:8080`: Maps port 8080 on the host to port 8080 in the container.
# `literary-jo:latest`: Specifies the Docker image to use.

echo "Container literary-jo-container is running with Go version $GO_VERSION and base image $BASE_IMAGE"
# Prints a confirmation message indicating the container is up and running, including the Go version and base image details.

#!/bin/bash
set -e

echo "Setting up Go development environment..."

# Update package lists
sudo apt-get update

# Install required system packages
sudo apt-get install -y wget curl git build-essential

# Install Go 1.24.4 (latest stable that meets the requirement)
GO_VERSION="1.24.4"
GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://golang.org/dl/${GO_TARBALL}"

echo "Installing Go ${GO_VERSION}..."
cd /tmp
wget -q "${GO_URL}"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "${GO_TARBALL}"

# Add Go to PATH in user's profile
echo 'export PATH=$PATH:/usr/local/go/bin' >> $HOME/.profile
echo 'export GOPATH=$HOME/go' >> $HOME/.profile
echo 'export PATH=$PATH:$GOPATH/bin' >> $HOME/.profile

# Set environment variables for current session
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Verify Go installation
go version

# Change to the project directory
cd /mnt/persist/workspace

# Download Go module dependencies
echo "Installing Go dependencies..."
go mod download
go mod tidy

# Verify the project can build
echo "Building project to verify setup..."
go build -o mcp-pdf-reader cmd/mcp-pdf-reader/main.go

echo "Setup completed successfully!"
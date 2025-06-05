#!/bin/bash

# Exit on error
set -e

PROGRAM_NAME="cURLDataExtractor" # Choose your desired program name
SOURCE_FILE="main.go"

# --- Define the Coder's Name ---
# Replace "Your Actual Coder Name" with your name or handle.
CODER_NAME="Ömer Aslan"

# --- Construct ldflags to include the Coder's Name ---
# This will set the value of the 'main.CoderName' variable in your Go program.
# Ensure you have 'var CoderName string' declared in your main package.
LDFLAGS_WITH_CODER_NAME="-X 'main.CoderName=${CODER_NAME}' -s -w"

# Create bin directory if it doesn't exist to store compiled binaries
mkdir -p ./bin

echo "Embedding Coder Name: ${CODER_NAME}"
echo ""

echo "Building for Windows (64-bit)..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="${LDFLAGS_WITH_CODER_NAME}" -o "./bin/${PROGRAM_NAME}_windows_amd64.exe" $SOURCE_FILE

echo "Building for Linux (64-bit)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS_WITH_CODER_NAME}" -o "./bin/${PROGRAM_NAME}_linux_amd64" $SOURCE_FILE

 echo "Building for macOS (64-bit Intel)..."
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="${LDFLAGS_WITH_CODER_NAME}" -o "./bin/${PROGRAM_NAME}_darwin_amd64" $SOURCE_FILE

echo "Building for macOS (64-bit ARM - Apple Silicon)..."
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="${LDFLAGS_WITH_CODER_NAME}" -o "./bin/${PROGRAM_NAME}_darwin_arm64" $SOURCE_FILE

echo ""
echo "Build complete. Executables are in the ./bin directory."
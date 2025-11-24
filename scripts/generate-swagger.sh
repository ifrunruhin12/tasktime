#!/bin/bash

# Generate Swagger documentation for TaskTime API

echo "Generating Swagger documentation..."

# Ensure swag is installed
if ! command -v swag &> /dev/null; then
    echo "swag not found in PATH. Installing..."
    go install github.com/swaggo/swag/cmd/swag@latest
    export PATH=$PATH:$(go env GOPATH)/bin
fi

# Generate docs
swag init -g cmd/server/main.go -o docs

if [ $? -eq 0 ]; then
    echo "✅ Swagger documentation generated successfully!"
    echo "📚 View at: http://localhost:8080/swagger/index.html (when server is running)"
else
    echo "❌ Failed to generate Swagger documentation"
    exit 1
fi

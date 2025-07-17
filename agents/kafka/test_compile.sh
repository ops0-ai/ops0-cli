#!/bin/bash

echo "🧪 Testing Kafka Agent Compilation"
echo "=================================="

# Navigate to the Kafka agent directory
cd "$(dirname "$0")"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed"
    exit 1
fi

echo "✅ Go version: $(go version)"
echo ""

# Initialize module if needed
if [ ! -f "go.mod" ]; then
    echo "📦 Initializing Go module..."
    go mod init kafka-agent
fi

# Download dependencies
echo "⬇️  Downloading dependencies..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "❌ Failed to download dependencies"
    exit 1
fi

echo "✅ Dependencies downloaded"
echo ""

# Test compilation of all files together
echo "🔨 Testing compilation..."
go build -o kafka-agent .
if [ $? -ne 0 ]; then
    echo "❌ Compilation failed"
    exit 1
fi

echo "✅ Compilation successful!"
echo ""

# Test the binary
echo "🧪 Testing binary..."
./kafka-agent --help
if [ $? -ne 0 ]; then
    echo "❌ Binary test failed"
    exit 1
fi

echo "✅ Binary works correctly!"
echo ""

echo "🎉 All tests passed! Kafka agent is ready to use."
echo ""
echo "📋 Available commands:"
echo "   ./kafka-agent --help"
echo "   ./kafka-agent producer --help"
echo "   ./kafka-agent consumer --help"
echo "   ./kafka-agent broker --help"
echo "   ./kafka-agent topic --help" 
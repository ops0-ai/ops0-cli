#!/bin/bash

# Kafka Agent Compilation Script
# This script shows how to compile the Kafka agent step by step

echo "🚀 Kafka Agent Compilation Guide"
echo "================================"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go first."
    echo "   Download from: https://golang.org/dl/"
    exit 1
fi

echo "✅ Go is installed: $(go version)"
echo ""

# Step 1: Navigate to the Kafka agent directory
echo "📁 Step 1: Navigating to Kafka agent directory..."
cd "$(dirname "$0")"
echo "   Current directory: $(pwd)"
echo ""

# Step 2: Initialize Go module (if not already done)
echo "📦 Step 2: Checking Go module..."
if [ ! -f "go.mod" ]; then
    echo "   Initializing Go module..."
    go mod init kafka-agent
else
    echo "   Go module already exists"
fi
echo ""

# Step 3: Download dependencies
echo "⬇️  Step 3: Downloading dependencies..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "❌ Failed to download dependencies"
    exit 1
fi
echo "✅ Dependencies downloaded successfully"
echo ""

# Step 4: Check for compilation errors in individual files
echo "🔍 Step 4: Checking individual files for syntax errors..."
echo "   Checking main.go..."
go build -o /dev/null main.go
if [ $? -ne 0 ]; then
    echo "❌ Error in main.go"
    exit 1
fi

echo "   Checking producer.go..."
go build -o /dev/null producer.go
if [ $? -ne 0 ]; then
    echo "❌ Error in producer.go"
    exit 1
fi

echo "   Checking consumer.go..."
go build -o /dev/null consumer.go
if [ $? -ne 0 ]; then
    echo "❌ Error in consumer.go"
    exit 1
fi

echo "   Checking broker.go..."
go build -o /dev/null broker.go
if [ $? -ne 0 ]; then
    echo "❌ Error in broker.go"
    exit 1
fi

echo "   Checking topic.go..."
go build -o /dev/null topic.go
if [ $? -ne 0 ]; then
    echo "❌ Error in topic.go"
    exit 1
fi

echo "✅ All individual files compile successfully"
echo ""

# Step 5: Compile the complete application
echo "🔨 Step 5: Compiling complete Kafka agent..."
go build -o kafka-agent .
if [ $? -ne 0 ]; then
    echo "❌ Failed to compile Kafka agent"
    exit 1
fi
echo "✅ Kafka agent compiled successfully!"
echo ""

# Step 6: Make the binary executable
echo "🔧 Step 6: Making binary executable..."
chmod +x kafka-agent
echo "✅ Binary is now executable"
echo ""

# Step 7: Test the binary
echo "🧪 Step 7: Testing the binary..."
./kafka-agent --help
if [ $? -ne 0 ]; then
    echo "❌ Binary test failed"
    exit 1
fi
echo "✅ Binary test successful!"
echo ""

echo "🎉 Compilation completed successfully!"
echo ""
echo "📋 Available commands:"
echo "   ./kafka-agent --help                    # Show help"
echo "   ./kafka-agent producer --help          # Producer commands"
echo "   ./kafka-agent consumer --help          # Consumer commands"
echo "   ./kafka-agent broker --help            # Broker commands"
echo "   ./kafka-agent topic --help             # Topic commands"
echo ""
echo "💡 Example usage:"
echo "   ./kafka-agent broker status"
echo "   ./kafka-agent topic list"
echo "   ./kafka-agent producer publish test-topic 'Hello World'"
echo "" 
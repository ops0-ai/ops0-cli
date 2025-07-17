# Kafka Agent Compilation Guide

This guide will help you compile the Kafka agent step by step.

## Prerequisites

1. **Go 1.19 or later** - Download from [golang.org/dl](https://golang.org/dl/)
2. **Git** - For version control (optional)

## Step-by-Step Compilation

### Step 1: Navigate to the Kafka Agent Directory

```bash
cd ops0-cli/agents/kafka
```

### Step 2: Check Go Installation

```bash
go version
```

You should see output like: `go version go1.19 linux/amd64`

### Step 3: Initialize Go Module (if not already done)

```bash
go mod init kafka-agent
```

### Step 4: Download Dependencies

```bash
go mod tidy
```

This will download all required dependencies (Cobra, Viper, etc.)

### Step 5: Compile the Application

```bash
go build -o kafka-agent .
```

### Step 6: Make the Binary Executable

```bash
chmod +x kafka-agent
```

### Step 7: Test the Binary

```bash
./kafka-agent --help
```

## Quick Compilation Script

You can use the provided script for quick compilation:

```bash
chmod +x test_compile.sh
./test_compile.sh
```

## File Structure

The Kafka agent consists of the following files:

- `main.go` - Main application entry point and configuration
- `types.go` - Shared types and helper functions
- `producer.go` - Producer management commands
- `consumer.go` - Consumer management commands
- `broker.go` - Broker management commands
- `topic.go` - Topic management commands
- `go.mod` - Go module definition
- `go.sum` - Dependency checksums (generated)

## Troubleshooting

### Common Issues

1. **"undefined: configureProducer" error**
   - Make sure all files are in the same directory
   - Check that all files have `package main` at the top

2. **"cannot find module" error**
   - Run `go mod tidy` to download dependencies
   - Check your internet connection

3. **"permission denied" error**
   - Run `chmod +x kafka-agent` to make the binary executable

4. **"command not found: go" error**
   - Install Go from [golang.org/dl](https://golang.org/dl/)
   - Add Go to your PATH

### Individual File Compilation

You can test individual files for syntax errors:

```bash
# Test main.go
go build -o /dev/null main.go

# Test producer.go
go build -o /dev/null producer.go

# Test consumer.go
go build -o /dev/null consumer.go

# Test broker.go
go build -o /dev/null broker.go

# Test topic.go
go build -o /dev/null topic.go

# Test types.go
go build -o /dev/null types.go
```

**Note**: Individual file compilation may show errors because some functions are defined in other files. This is normal - the important thing is that the complete application compiles successfully.

## Usage Examples

After successful compilation, you can use the Kafka agent:

```bash
# Show help
./kafka-agent --help

# Producer commands
./kafka-agent producer config --topic test-topic --acks all
./kafka-agent producer publish test-topic "Hello World"

# Consumer commands
./kafka-agent consumer config --topic test-topic --group-id test-group
./kafka-agent consumer consume test-topic --max-messages 5

# Broker commands
./kafka-agent broker status
./kafka-agent broker health

# Topic commands
./kafka-agent topic list
./kafka-agent topic create new-topic --partitions 3
```

## Integration with Main CLI

The Kafka agent is designed to integrate with your main CLI tool. You can:

1. Use it as a standalone tool
2. Integrate it with your main CLI by calling the binary
3. Import the functions into your main CLI code

## Next Steps

1. Test the compiled binary with various commands
2. Customize the configuration for your Kafka cluster
3. Integrate with your main CLI tool
4. Add real Kafka client integration (e.g., Sarama library) 
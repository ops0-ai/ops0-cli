# Kafka Agent - Comprehensive Kafka Management Tool

A powerful command-line tool for managing Apache Kafka clusters, producers, consumers, brokers, and topics. Built with Go and the Cobra CLI framework, this agent provides a comprehensive interface for Kafka operations with support for configuration management, health monitoring, and real-time status tracking.

## 🚀 Features

- **Producer Management**: Configure and manage Kafka producers with customizable settings
- **Consumer Management**: Set up consumers, subscribe to topics, and consume messages
- **Broker Monitoring**: Real-time broker status, health checks, and metrics
- **Topic Management**: Create, delete, list, and configure Kafka topics
- **Configuration Management**: Persistent configuration with YAML support
- **Security Support**: SSL/TLS and SASL authentication
- **Interactive Mode**: Natural language command processing
- **Comprehensive Logging**: Detailed operation logging and monitoring

## 📋 Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Commands Reference](#commands-reference)
- [Usage Examples](#usage-examples)
- [Integration](#integration)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

## 🛠️ Installation

### Prerequisites

- Go 1.19 or later
- Access to a Kafka cluster
- Git (optional)

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd ops0-cli/agents/kafka

# Build the application
go build -o kafka-agent .

# Make executable
chmod +x kafka-agent

# Test installation
./kafka-agent --help
```

### Quick Build Script

```bash
chmod +x test_compile.sh
./test_compile.sh
```

## 🚀 Quick Start

### Basic Usage

```bash
# Show help
./kafka-agent --help

# Check broker status
./kafka-agent broker status

# List topics
./kafka-agent topic list

# Configure a producer
./kafka-agent producer config --topic my-topic --acks all

# Publish a message
./kafka-agent producer publish my-topic "Hello Kafka!"

# Configure a consumer
./kafka-agent consumer config --topic my-topic --group-id my-group

# Consume messages
./kafka-agent consumer consume my-topic --max-messages 10
```

## ⚙️ Configuration

### Configuration File

The agent supports configuration via YAML files. Default location: `$HOME/.kafka-agent.yaml`

```yaml
# ~/.kafka-agent.yaml
brokers:
  - "localhost:9092"
  - "localhost:9093"
security_protocol: "PLAINTEXT"
sasl_mechanism: ""
username: ""
password: ""
ssl_ca_file: ""
ssl_cert_file: ""
ssl_key_file: ""
```

### Environment Variables

```bash
export KAFKA_BROKERS="localhost:9092,localhost:9093"
export KAFKA_SECURITY_PROTOCOL="PLAINTEXT"
export KAFKA_SASL_MECHANISM="PLAIN"
export KAFKA_USERNAME="your-username"
export KAFKA_PASSWORD="your-password"
```

### Command Line Flags

```bash
./kafka-agent --brokers "localhost:9092,localhost:9093" \
              --security-protocol "PLAINTEXT" \
              --sasl-mechanism "PLAIN" \
              --username "user" \
              --password "pass"
```

## 📚 Commands Reference

### Global Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Config file path | `$HOME/.kafka-agent.yaml` |
| `--brokers` | Kafka broker addresses | `localhost:9092` |
| `--security-protocol` | Security protocol | `PLAINTEXT` |
| `--sasl-mechanism` | SASL mechanism | `` |
| `--username` | SASL username | `` |
| `--password` | SASL password | `` |

### Producer Commands

#### `producer config` - Configure Producer Settings

```bash
./kafka-agent producer config [flags]
```

**Flags:**
- `--topic` - Topic name
- `--acks` - Number of acknowledgments (0, 1, all) [default: "all"]
- `--retries` - Number of retries [default: 3]
- `--batch-size` - Batch size in bytes [default: 16384]
- `--linger-ms` - Linger time in milliseconds [default: 0]
- `--compression` - Compression type (none, gzip, snappy, lz4) [default: "none"]
- `--key-serializer` - Key serializer class
- `--value-serializer` - Value serializer class

**Example:**
```bash
./kafka-agent producer config \
  --topic events \
  --acks all \
  --retries 5 \
  --batch-size 32768 \
  --compression snappy
```

#### `producer publish` - Publish Message to Topic

```bash
./kafka-agent producer publish <topic> <message> [flags]
```

**Flags:**
- `--key` - Message key
- `--partition` - Partition number (optional)
- `--headers` - Message headers as JSON

**Examples:**
```bash
# Basic message
./kafka-agent producer publish events "Hello World"

# With key
./kafka-agent producer publish events "Hello World" --key "user-123"

# With headers
./kafka-agent producer publish events "Hello World" \
  --key "user-123" \
  --headers '{"source": "kafka-agent", "version": "1.0"}'

# To specific partition
./kafka-agent producer publish events "Hello World" --partition 0
```

#### `producer status` - Show Producer Status

```bash
./kafka-agent producer status
```

**Output:**
```
📊 Producer Status:
   Topic: events
   Acks: all
   Retries: 5
   Batch Size: 32768 bytes
   Linger: 0 ms
   Compression: snappy
   Key Serializer: org.apache.kafka.common.serialization.StringSerializer
   Value Serializer: org.apache.kafka.common.serialization.StringSerializer
   Connection: ✅ Connected
```

### Consumer Commands

#### `consumer config` - Configure Consumer Settings

```bash
./kafka-agent consumer config [flags]
```

**Flags:**
- `--topic` - Topic name
- `--group-id` - Consumer group ID
- `--auto-offset-reset` - Auto offset reset (earliest, latest) [default: "earliest"]
- `--enable-auto-commit` - Enable auto commit [default: true]
- `--auto-commit-interval` - Auto commit interval in milliseconds [default: 5000]
- `--session-timeout` - Session timeout in milliseconds [default: 30000]
- `--max-poll-records` - Max poll records [default: 500]
- `--key-deserializer` - Key deserializer class
- `--value-deserializer` - Value deserializer class

**Example:**
```bash
./kafka-agent consumer config \
  --topic events \
  --group-id event-processors \
  --auto-offset-reset earliest \
  --enable-auto-commit true \
  --session-timeout 45000
```

#### `consumer subscribe` - Subscribe to Topic

```bash
./kafka-agent consumer subscribe <topic>
```

**Example:**
```bash
./kafka-agent consumer subscribe events
```

#### `consumer consume` - Consume Messages

```bash
./kafka-agent consumer consume <topic> [flags]
```

**Flags:**
- `--timeout` - Consume timeout in milliseconds [default: 5000]
- `--max-messages` - Maximum number of messages to consume [default: 10]
- `--follow` - Follow mode (continuous consumption) [default: false]

**Examples:**
```bash
# Consume 5 messages
./kafka-agent consumer consume events --max-messages 5

# Follow mode (continuous)
./kafka-agent consumer consume events --follow

# With custom timeout
./kafka-agent consumer consume events --timeout 10000 --max-messages 20
```

#### `consumer status` - Show Consumer Status

```bash
./kafka-agent consumer status
```

#### `consumer groups` - List Consumer Groups

```bash
./kafka-agent consumer groups
```

**Output:**
```
📋 Consumer Groups:

   Group ID: event-processors
   State: Stable
   Members: 3
   Topics: [events, notifications]
   Last Activity: 2024-01-15T10:30:00Z

   Group ID: analytics-consumers
   State: Empty
   Members: 0
   Topics: [analytics]
   Last Activity: 2024-01-15T09:15:00Z
```

### Broker Commands

#### `broker status` - Show Broker Status

```bash
./kafka-agent broker status
```

**Output:**
```
📊 Kafka Broker Status:

🖥️  Broker 0 (localhost:9092):
   Status: 🟢 Online
   Rack: rack-1
   Leader: true
   Replicas: 3
   Uptime: 2 days, 5 hours, 30 minutes
   Version: 3.5.1

🖥️  Broker 1 (localhost:9093):
   Status: 🟢 Online
   Rack: rack-1
   Leader: false
   Replicas: 2
   Uptime: 2 days, 5 hours, 25 minutes
   Version: 3.5.1

📈 Summary:
   Total Brokers: 2
   Online: 2
   Offline: 0
   Health: 🟢 Excellent
```

#### `broker health` - Check Broker Health

```bash
./kafka-agent broker health
```

**Output:**
```
🏥 Kafka Broker Health Check:

🖥️  Checking Broker 0 (localhost:9092)...
   TCP Connectivity: ✅ PASS
   Kafka Protocol: ✅ PASS
   Disk Space: ✅ PASS
   Memory Usage: ✅ PASS
   CPU Usage: ✅ PASS
   Network Latency: ✅ PASS

🖥️  Checking Broker 1 (localhost:9093)...
   TCP Connectivity: ✅ PASS
   Kafka Protocol: ✅ PASS
   Disk Space: ❌ FAIL
      Error: Disk usage above 90%
   Memory Usage: ✅ PASS
   CPU Usage: ✅ PASS
   Network Latency: ✅ PASS

✅ Health check completed!
```

#### `broker config` - Show Broker Configuration

```bash
./kafka-agent broker config
```

#### `broker metrics` - Show Broker Metrics

```bash
./kafka-agent broker metrics
```

**Output:**
```
📈 Kafka Broker Metrics:

🖥️  Broker 0 (localhost:9092):
   Messages/sec: 1250.50
   Bytes/sec: 2048576.00
   Active Controllers: 1
   Offline Partitions: 0
   Under Replicated Partitions: 0
   Total Time: 45.20 ms
   Request Queue Size: 5
   Response Queue Size: 3
   Network Processor Avg Idle: 85.50%
   Request Handler Avg Idle: 78.20%
```

#### `broker connect` - Test Broker Connectivity

```bash
./kafka-agent broker connect
```

**Output:**
```
🔌 Testing Kafka Broker Connectivity:

🖥️  Testing connection to localhost:9092...
   ✅ TCP connection successful
   ✅ Kafka protocol test successful

🖥️  Testing connection to localhost:9093...
   ✅ TCP connection successful
   ✅ Kafka protocol test successful

✅ Connectivity test completed!
```

### Topic Commands

#### `topic list` - List All Topics

```bash
./kafka-agent topic list
```

**Output:**
```
📋 Kafka Topics:

📄 Topic: events
   Partitions: 3
   Replicas: 2
   Status: 🟢 Active

📄 Topic: notifications
   Partitions: 1
   Replicas: 1
   Status: 🟢 Active

📄 Topic: analytics
   Partitions: 5
   Replicas: 3
   Status: 🟢 Active

📊 Summary: 3 topics found
```

#### `topic create` - Create New Topic

```bash
./kafka-agent topic create <topic> [flags]
```

**Flags:**
- `--partitions` - Number of partitions [default: 1]
- `--replication-factor` - Replication factor [default: 1]
- `--cleanup-policy` - Cleanup policy (delete, compact) [default: "delete"]
- `--retention-ms` - Retention time in milliseconds [default: 604800000]
- `--segment-bytes` - Segment size in bytes [default: 1073741824]

**Examples:**
```bash
# Basic topic
./kafka-agent topic create my-topic

# With custom settings
./kafka-agent topic create events \
  --partitions 6 \
  --replication-factor 3 \
  --cleanup-policy delete \
  --retention-ms 86400000

# Compacted topic
./kafka-agent topic create user-profiles \
  --partitions 3 \
  --replication-factor 2 \
  --cleanup-policy compact
```

#### `topic delete` - Delete Topic

```bash
./kafka-agent topic delete <topic>
```

**Example:**
```bash
./kafka-agent topic delete old-topic
```

#### `topic describe` - Describe Topic

```bash
./kafka-agent topic describe <topic>
```

**Output:**
```
📄 Topic Details: events
   Name: events
   Partitions: 3
   Replicas: 2

📊 Partition Information:
   Partition 0:
     Leader: 0
     Replicas: [0, 1]
     ISR: [0, 1]
     Status: Online

   Partition 1:
     Leader: 1
     Replicas: [1, 2]
     ISR: [1, 2]
     Status: Online

   Partition 2:
     Leader: 0
     Replicas: [0, 2]
     ISR: [0, 2]
     Status: Online

⚙️  Configuration:
   cleanup.policy: delete
   retention.ms: 604800000
   segment.bytes: 1073741824
```

#### `topic config` - Show Topic Configuration

```bash
./kafka-agent topic config <topic>
```

## 💡 Usage Examples

### Complete Workflow Example

```bash
# 1. Check cluster health
./kafka-agent broker health

# 2. List existing topics
./kafka-agent topic list

# 3. Create a new topic
./kafka-agent topic create user-events \
  --partitions 6 \
  --replication-factor 3

# 4. Configure producer
./kafka-agent producer config \
  --topic user-events \
  --acks all \
  --compression snappy

# 5. Publish messages
./kafka-agent producer publish user-events "User logged in" --key "user-123"
./kafka-agent producer publish user-events "User made purchase" --key "user-456"

# 6. Configure consumer
./kafka-agent consumer config \
  --topic user-events \
  --group-id event-processors \
  --auto-offset-reset earliest

# 7. Subscribe and consume
./kafka-agent consumer subscribe user-events
./kafka-agent consumer consume user-events --max-messages 10

# 8. Monitor status
./kafka-agent broker status
./kafka-agent consumer groups
```

### Security Configuration Example

```bash
# SSL/TLS Configuration
./kafka-agent --brokers "ssl-broker:9093" \
  --security-protocol "SSL" \
  --ssl-ca-file "/path/to/ca.pem" \
  --ssl-cert-file "/path/to/cert.pem" \
  --ssl-key-file "/path/to/key.pem" \
  broker status

# SASL/PLAIN Authentication
./kafka-agent --brokers "sasl-broker:9092" \
  --security-protocol "SASL_PLAINTEXT" \
  --sasl-mechanism "PLAIN" \
  --username "kafka-user" \
  --password "kafka-password" \
  topic list
```

### Monitoring and Alerting

```bash
# Check cluster health periodically
while true; do
  ./kafka-agent broker health
  sleep 300  # Check every 5 minutes
done

# Monitor topic growth
./kafka-agent topic describe large-topic | grep "Partitions"

# Check consumer lag
./kafka-agent consumer groups | grep "Stable"
```

## 🔧 Integration

### Integration with Main CLI

The Kafka agent can be integrated with your main CLI tool:

```bash
# Call from main CLI
ops0 -m "show Kafka broker status"
ops0 -m "publish message to topic events"
ops0 -m "list all Kafka topics"
```

### API Integration

```go
// Example Go code to use the Kafka agent
agent := NewKafkaIntegration()

// Execute Kafka command
command := &KafkaCommand{
    Operation: "broker_status",
}
err := agent.ExecuteKafkaCommand(command)
```

### Configuration Integration

```yaml
# Main CLI configuration
kafka:
  enabled: true
  brokers:
    - "localhost:9092"
  security_protocol: "PLAINTEXT"
  default_topic: "events"
```

## 🐛 Troubleshooting

### Common Issues

#### 1. Connection Errors

**Error:** `❌ Could not connect to Kafka cluster`

**Solutions:**
- Check broker addresses: `./kafka-agent --brokers "broker1:9092,broker2:9092"`
- Verify network connectivity: `./kafka-agent broker connect`
- Check security settings for SSL/SASL

#### 2. Permission Errors

**Error:** `permission denied`

**Solution:**
```bash
chmod +x kafka-agent
```

#### 3. Configuration Issues

**Error:** `no producer configuration found`

**Solution:**
```bash
./kafka-agent producer config --topic my-topic
```

#### 4. Topic Not Found

**Error:** `topic 'my-topic' not found`

**Solution:**
```bash
# List existing topics
./kafka-agent topic list

# Create the topic if needed
./kafka-agent topic create my-topic
```

### Debug Mode

Enable verbose logging:

```bash
# Set debug environment variable
export KAFKA_DEBUG=true

# Run commands with debug output
./kafka-agent broker status
```

### Health Check Script

```bash
#!/bin/bash
# health_check.sh

echo "🔍 Kafka Cluster Health Check"
echo "============================="

# Check broker connectivity
echo "1. Testing broker connectivity..."
./kafka-agent broker connect

# Check broker health
echo "2. Checking broker health..."
./kafka-agent broker health

# List topics
echo "3. Listing topics..."
./kafka-agent topic list

# Check consumer groups
echo "4. Checking consumer groups..."
./kafka-agent consumer groups

echo "✅ Health check completed!"
```

## 🚀 Development

### Project Structure

```
ops0-cli/agents/kafka/
├── main.go           # Main application entry point
├── types.go          # Shared types and helper functions
├── producer.go       # Producer management commands
├── consumer.go       # Consumer management commands
├── broker.go         # Broker management commands
├── topic.go          # Topic management commands
├── go.mod            # Go module definition
├── go.sum            # Dependency checksums
├── README.md         # This file
├── COMPILATION_GUIDE.md  # Compilation instructions
├── test_compile.sh   # Test compilation script
└── compile.sh        # Full compilation script
```

### Adding New Commands

1. Create a new command file (e.g., `cluster.go`)
2. Define the command structure using Cobra
3. Add the command to `main.go`
4. Update this README with documentation

### Testing

```bash
# Run compilation tests
./test_compile.sh

# Test individual components
go test ./...

# Integration tests
./kafka-agent --help
./kafka-agent broker status
./kafka-agent topic list
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Update documentation
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🤝 Support

- **Documentation:** [README.md](README.md)
- **Issues:** [GitHub Issues](https://github.com/your-repo/issues)
- **Discussions:** [GitHub Discussions](https://github.com/your-repo/discussions)

## 🔄 Version History

- **v1.0.0** - Initial release with basic producer/consumer/broker/topic management
- **v1.1.0** - Added security support (SSL/SASL)
- **v1.2.0** - Enhanced monitoring and metrics
- **v1.3.0** - Added configuration management and persistence

---

**Happy Kafka Management! 🚀** 
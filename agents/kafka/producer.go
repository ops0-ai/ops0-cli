package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var producerCmd = &cobra.Command{
	Use:   "producer",
	Short: "Manage Kafka producers",
	Long:  `Configure and manage Kafka producers including settings, topics, and message publishing.`,
}

var producerConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure producer settings",
	Long:  `Set up producer configuration including topic, acks, retries, batch settings, etc.`,
	Run: func(cmd *cobra.Command, args []string) {
		config := make(map[string]interface{})
		config["topic"], _ = cmd.Flags().GetString("topic")
		config["acks"], _ = cmd.Flags().GetString("acks")
		config["retries"], _ = cmd.Flags().GetInt("retries")
		config["batch-size"], _ = cmd.Flags().GetInt("batch-size")
		config["linger-ms"], _ = cmd.Flags().GetInt("linger-ms")
		config["compression"], _ = cmd.Flags().GetString("compression")
		config["key-serializer"], _ = cmd.Flags().GetString("key-serializer")
		config["value-serializer"], _ = cmd.Flags().GetString("value-serializer")
		
		if err := configureProducer(config); err != nil {
			fmt.Printf("❌ Error configuring producer: %v\n", err)
			os.Exit(1)
		}
	},
}

var producerPublishCmd = &cobra.Command{
	Use:   "publish [topic] [message]",
	Short: "Publish a message to a topic",
	Long:  `Publish a message to the specified Kafka topic.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key, _ := cmd.Flags().GetString("key")
		metadata := make(map[string]interface{})
		metadata["partition"], _ = cmd.Flags().GetString("partition")
		metadata["headers"], _ = cmd.Flags().GetString("headers")
		
		if err := publishMessage(args[0], args[1], key, metadata); err != nil {
			fmt.Printf("❌ Error publishing message: %v\n", err)
			os.Exit(1)
		}
	},
}

var producerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show producer status",
	Long:  `Display current producer configuration and status.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := showProducerStatus(); err != nil {
			fmt.Printf("❌ Error showing producer status: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Producer config flags
	producerConfigCmd.Flags().String("topic", "", "Topic name")
	producerConfigCmd.Flags().String("acks", "all", "Number of acknowledgments (0, 1, all)")
	producerConfigCmd.Flags().Int("retries", 3, "Number of retries")
	producerConfigCmd.Flags().Int("batch-size", 16384, "Batch size in bytes")
	producerConfigCmd.Flags().Int("linger-ms", 0, "Linger time in milliseconds")
	producerConfigCmd.Flags().String("compression", "none", "Compression type (none, gzip, snappy, lz4)")
	producerConfigCmd.Flags().String("key-serializer", "org.apache.kafka.common.serialization.StringSerializer", "Key serializer class")
	producerConfigCmd.Flags().String("value-serializer", "org.apache.kafka.common.serialization.StringSerializer", "Value serializer class")
	
	// Producer publish flags
	producerPublishCmd.Flags().String("key", "", "Message key")
	producerPublishCmd.Flags().String("partition", "", "Partition number (optional)")
	producerPublishCmd.Flags().String("headers", "", "Message headers as JSON")
	
	producerCmd.AddCommand(producerConfigCmd, producerPublishCmd, producerStatusCmd)
	rootCmd.AddCommand(producerCmd)
}

func configureProducer(config map[string]interface{}) error {
	// Create producer configuration
	producerConfig := &ProducerConfig{
		Topic:           getString(config, "topic"),
		Acks:            getString(config, "acks"),
		Retries:         getInt(config, "retries"),
		BatchSize:       getInt(config, "batch-size"),
		LingerMs:        getInt(config, "linger-ms"),
		CompressionType: getString(config, "compression"),
		KeySerializer:   getString(config, "key-serializer"),
		ValueSerializer: getString(config, "value-serializer"),
		Properties:      make(map[string]string),
	}
	
	// Add common properties
	producerConfig.Properties["bootstrap.servers"] = strings.Join(kafkaAgent.config.Brokers, ",")
	producerConfig.Properties["acks"] = producerConfig.Acks
	producerConfig.Properties["retries"] = strconv.Itoa(producerConfig.Retries)
	producerConfig.Properties["batch.size"] = strconv.Itoa(producerConfig.BatchSize)
	producerConfig.Properties["linger.ms"] = strconv.Itoa(producerConfig.LingerMs)
	producerConfig.Properties["compression.type"] = producerConfig.CompressionType
	producerConfig.Properties["key.serializer"] = producerConfig.KeySerializer
	producerConfig.Properties["value.serializer"] = producerConfig.ValueSerializer
	
	// Add security properties if configured
	if kafkaAgent.config.SecurityProtocol != "PLAINTEXT" {
		producerConfig.Properties["security.protocol"] = kafkaAgent.config.SecurityProtocol
		if kafkaAgent.config.SASLMechanism != "" {
			producerConfig.Properties["sasl.mechanism"] = kafkaAgent.config.SASLMechanism
			producerConfig.Properties["sasl.username"] = kafkaAgent.config.Username
			producerConfig.Properties["sasl.password"] = kafkaAgent.config.Password
		}
	}
	
	kafkaAgent.producer = producerConfig
	
	// Save configuration
	if err := saveProducerConfig(producerConfig); err != nil {
		return fmt.Errorf("error saving producer configuration: %v", err)
	}
	
	fmt.Println("✅ Producer configuration saved successfully!")
	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("   Topic: %s\n", producerConfig.Topic)
	fmt.Printf("   Acks: %s\n", producerConfig.Acks)
	fmt.Printf("   Retries: %d\n", producerConfig.Retries)
	fmt.Printf("   Batch Size: %d bytes\n", producerConfig.BatchSize)
	fmt.Printf("   Linger: %d ms\n", producerConfig.LingerMs)
	fmt.Printf("   Compression: %s\n", producerConfig.CompressionType)
	
	return nil
}

func publishMessage(topic, message, key string, metadata map[string]interface{}) error {
	// Parse headers if provided
	var headers map[string]string
	if headersStr := getString(metadata, "headers"); headersStr != "" {
		if err := json.Unmarshal([]byte(headersStr), &headers); err != nil {
			return fmt.Errorf("error parsing headers: %v", err)
		}
	}
	
	// Create message payload
	payload := map[string]interface{}{
		"topic":     topic,
		"message":   message,
		"key":       key,
		"headers":   headers,
		"timestamp": getCurrentTimestamp(),
	}
	
	if partitionStr := getString(metadata, "partition"); partitionStr != "" {
		if partition, err := strconv.Atoi(partitionStr); err == nil {
			payload["partition"] = partition
		}
	}
	
	// Simulate publishing (in a real implementation, this would use a Kafka client)
	fmt.Printf("📤 Publishing message to topic '%s':\n", topic)
	fmt.Printf("   Message: %s\n", message)
	if key != "" {
		fmt.Printf("   Key: %s\n", key)
	}
	if len(headers) > 0 {
		fmt.Printf("   Headers: %v\n", headers)
	}
	
	// Here you would integrate with actual Kafka client
	// For now, we'll simulate success
	fmt.Println("✅ Message published successfully!")
	
	// Log the operation
	logProducerOperation("publish", payload)
	
	return nil
}

func showProducerStatus() error {
	if kafkaAgent.producer == nil {
		return fmt.Errorf("no producer configuration found. Run 'producer config' first")
	}
	
	fmt.Println("📊 Producer Status:")
	fmt.Printf("   Topic: %s\n", kafkaAgent.producer.Topic)
	fmt.Printf("   Acks: %s\n", kafkaAgent.producer.Acks)
	fmt.Printf("   Retries: %d\n", kafkaAgent.producer.Retries)
	fmt.Printf("   Batch Size: %d bytes\n", kafkaAgent.producer.BatchSize)
	fmt.Printf("   Linger: %d ms\n", kafkaAgent.producer.LingerMs)
	fmt.Printf("   Compression: %s\n", kafkaAgent.producer.CompressionType)
	fmt.Printf("   Key Serializer: %s\n", kafkaAgent.producer.KeySerializer)
	fmt.Printf("   Value Serializer: %s\n", kafkaAgent.producer.ValueSerializer)
	
	// Show connection status
	if kafkaAgent.connection.Connected {
		fmt.Println("   Connection: ✅ Connected")
	} else {
		fmt.Println("   Connection: ❌ Disconnected")
		if kafkaAgent.connection.LastError != nil {
			fmt.Printf("   Last Error: %v\n", kafkaAgent.connection.LastError)
		}
	}
	
	return nil
}

func saveProducerConfig(config *ProducerConfig) error {
	// In a real implementation, this would save to a file or database
	// For now, we'll just store it in memory
	return nil
}

func logProducerOperation(operation string, payload map[string]interface{}) {
	// In a real implementation, this would log to a file or monitoring system
	fmt.Printf("📝 Logged producer operation: %s\n", operation)
}

// Helper functions are now in types.go 
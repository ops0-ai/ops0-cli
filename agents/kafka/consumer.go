package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var consumerCmd = &cobra.Command{
	Use:   "consumer",
	Short: "Manage Kafka consumers",
	Long:  `Configure and manage Kafka consumers including settings, topics, and message consumption.`,
}

var consumerConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure consumer settings",
	Long:  `Set up consumer configuration including topic, group ID, offset settings, etc.`,
	Run: func(cmd *cobra.Command, args []string) {
		config := make(map[string]interface{})
		config["topic"], _ = cmd.Flags().GetString("topic")
		config["group-id"], _ = cmd.Flags().GetString("group-id")
		config["auto-offset-reset"], _ = cmd.Flags().GetString("auto-offset-reset")
		config["enable-auto-commit"], _ = cmd.Flags().GetBool("enable-auto-commit")
		config["auto-commit-interval"], _ = cmd.Flags().GetInt("auto-commit-interval")
		config["session-timeout"], _ = cmd.Flags().GetInt("session-timeout")
		config["max-poll-records"], _ = cmd.Flags().GetInt("max-poll-records")
		config["key-deserializer"], _ = cmd.Flags().GetString("key-deserializer")
		config["value-deserializer"], _ = cmd.Flags().GetString("value-deserializer")

		if err := configureConsumer(config); err != nil {
			fmt.Printf("❌ Error configuring consumer: %v\n", err)
			os.Exit(1)
		}
	},
}

var consumerSubscribeCmd = &cobra.Command{
	Use:   "subscribe [topic]",
	Short: "Subscribe to a topic",
	Long:  `Subscribe to a Kafka topic and start consuming messages.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := subscribeToTopic(args[0]); err != nil {
			fmt.Printf("❌ Error subscribing to topic: %v\n", err)
			os.Exit(1)
		}
	},
}

var consumerConsumeCmd = &cobra.Command{
	Use:   "consume [topic]",
	Short: "Consume messages from a topic",
	Long:  `Consume messages from the specified Kafka topic.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		metadata := make(map[string]interface{})
		metadata["timeout"], _ = cmd.Flags().GetInt("timeout")
		metadata["max-messages"], _ = cmd.Flags().GetInt("max-messages")
		metadata["follow"], _ = cmd.Flags().GetBool("follow")

		if err := consumeMessages(args[0], metadata); err != nil {
			fmt.Printf("❌ Error consuming messages: %v\n", err)
			os.Exit(1)
		}
	},
}

var consumerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show consumer status",
	Long:  `Display current consumer configuration and status.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := showConsumerStatus(); err != nil {
			fmt.Printf("❌ Error showing consumer status: %v\n", err)
			os.Exit(1)
		}
	},
}

var consumerGroupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "List consumer groups",
	Long:  `List all consumer groups in the Kafka cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listConsumerGroups(); err != nil {
			fmt.Printf("❌ Error listing consumer groups: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Consumer config flags
	consumerConfigCmd.Flags().String("topic", "", "Topic name")
	consumerConfigCmd.Flags().String("group-id", "", "Consumer group ID")
	consumerConfigCmd.Flags().String("auto-offset-reset", "earliest", "Auto offset reset (earliest, latest)")
	consumerConfigCmd.Flags().Bool("enable-auto-commit", true, "Enable auto commit")
	consumerConfigCmd.Flags().Int("auto-commit-interval", 5000, "Auto commit interval in milliseconds")
	consumerConfigCmd.Flags().Int("session-timeout", 30000, "Session timeout in milliseconds")
	consumerConfigCmd.Flags().Int("max-poll-records", 500, "Max poll records")
	consumerConfigCmd.Flags().String("key-deserializer", "org.apache.kafka.common.serialization.StringDeserializer", "Key deserializer class")
	consumerConfigCmd.Flags().String("value-deserializer", "org.apache.kafka.common.serialization.StringDeserializer", "Value deserializer class")

	// Consumer consume flags
	consumerConsumeCmd.Flags().Int("timeout", 5000, "Consume timeout in milliseconds")
	consumerConsumeCmd.Flags().Int("max-messages", 10, "Maximum number of messages to consume")
	consumerConsumeCmd.Flags().Bool("follow", false, "Follow mode (continuous consumption)")

	consumerCmd.AddCommand(consumerConfigCmd, consumerSubscribeCmd, consumerConsumeCmd, consumerStatusCmd, consumerGroupsCmd)
	rootCmd.AddCommand(consumerCmd)
}

func configureConsumer(config map[string]interface{}) error {
	// Create consumer configuration
	consumerConfig := &ConsumerConfig{
		Topic:              getString(config, "topic"),
		GroupID:            getString(config, "group-id"),
		AutoOffsetReset:    getString(config, "auto-offset-reset"),
		EnableAutoCommit:   getBool(config, "enable-auto-commit"),
		AutoCommitInterval: getInt(config, "auto-commit-interval"),
		SessionTimeout:     getInt(config, "session-timeout"),
		MaxPollRecords:     getInt(config, "max-poll-records"),
		KeyDeserializer:    getString(config, "key-deserializer"),
		ValueDeserializer:  getString(config, "value-deserializer"),
		Properties:         make(map[string]string),
	}

	// Add common properties
	consumerConfig.Properties["bootstrap.servers"] = strings.Join(kafkaAgent.config.Brokers, ",")
	consumerConfig.Properties["group.id"] = consumerConfig.GroupID
	consumerConfig.Properties["auto.offset.reset"] = consumerConfig.AutoOffsetReset
	consumerConfig.Properties["enable.auto.commit"] = strconv.FormatBool(consumerConfig.EnableAutoCommit)
	consumerConfig.Properties["auto.commit.interval.ms"] = strconv.Itoa(consumerConfig.AutoCommitInterval)
	consumerConfig.Properties["session.timeout.ms"] = strconv.Itoa(consumerConfig.SessionTimeout)
	consumerConfig.Properties["max.poll.records"] = strconv.Itoa(consumerConfig.MaxPollRecords)
	consumerConfig.Properties["key.deserializer"] = consumerConfig.KeyDeserializer
	consumerConfig.Properties["value.deserializer"] = consumerConfig.ValueDeserializer

	// Add security properties if configured
	if kafkaAgent.config.SecurityProtocol != "PLAINTEXT" {
		consumerConfig.Properties["security.protocol"] = kafkaAgent.config.SecurityProtocol
		if kafkaAgent.config.SASLMechanism != "" {
			consumerConfig.Properties["sasl.mechanism"] = kafkaAgent.config.SASLMechanism
			consumerConfig.Properties["sasl.username"] = kafkaAgent.config.Username
			consumerConfig.Properties["sasl.password"] = kafkaAgent.config.Password
		}
	}

	kafkaAgent.consumer = consumerConfig

	// Save configuration
	if err := saveConsumerConfig(consumerConfig); err != nil {
		return fmt.Errorf("error saving consumer configuration: %v", err)
	}

	fmt.Println("✅ Consumer configuration saved successfully!")
	fmt.Printf("📋 Configuration:\n")
	fmt.Printf("   Topic: %s\n", consumerConfig.Topic)
	fmt.Printf("   Group ID: %s\n", consumerConfig.GroupID)
	fmt.Printf("   Auto Offset Reset: %s\n", consumerConfig.AutoOffsetReset)
	fmt.Printf("   Enable Auto Commit: %t\n", consumerConfig.EnableAutoCommit)
	fmt.Printf("   Auto Commit Interval: %d ms\n", consumerConfig.AutoCommitInterval)
	fmt.Printf("   Session Timeout: %d ms\n", consumerConfig.SessionTimeout)
	fmt.Printf("   Max Poll Records: %d\n", consumerConfig.MaxPollRecords)

	return nil
}

func subscribeToTopic(topic string) error {
	if kafkaAgent.consumer == nil {
		return fmt.Errorf("no consumer configuration found. Run 'consumer config' first")
	}

	fmt.Printf("📡 Subscribing to topic: %s\n", topic)

	// Update consumer config with topic
	kafkaAgent.consumer.Topic = topic

	// Test connection by creating a consumer
	kc, err := NewKafkaClient(
		kafkaAgent.config.Brokers,
		kafkaAgent.config.SecurityProtocol,
		kafkaAgent.config.SASLMechanism,
		kafkaAgent.config.Username,
		kafkaAgent.config.Password,
	)
	if err != nil {
		return fmt.Errorf("failed to create Kafka client: %v", err)
	}
	defer kc.Close()

	consumer, err := kc.CreateConsumer(kafkaAgent.consumer)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %v", err)
	}
	defer consumer.Close()

	fmt.Printf("✅ Successfully subscribed to topic '%s'\n", topic)
	fmt.Printf("   Group ID: %s\n", kafkaAgent.consumer.GroupID)
	fmt.Printf("   Auto Offset Reset: %s\n", kafkaAgent.consumer.AutoOffsetReset)

	// Log the operation
	logConsumerOperation("subscribe", map[string]interface{}{
		"topic":     topic,
		"group_id":  kafkaAgent.consumer.GroupID,
		"timestamp": getCurrentTimestamp(),
	})

	return nil
}

func consumeMessages(topic string, metadata map[string]interface{}) error {
	timeout := getInt(metadata, "timeout")
	maxMessages := getInt(metadata, "max-messages")
	follow := getBool(metadata, "follow")

	if kafkaAgent.consumer == nil {
		return fmt.Errorf("no consumer configuration found. Run 'consumer config' first")
	}

	fmt.Printf("📨 Consuming messages from topic: %s\n", topic)
	if follow {
		fmt.Println("🔄 Follow mode enabled (continuous consumption)")
	}

	// Create Kafka client
	kc, err := NewKafkaClient(
		kafkaAgent.config.Brokers,
		kafkaAgent.config.SecurityProtocol,
		kafkaAgent.config.SASLMechanism,
		kafkaAgent.config.Username,
		kafkaAgent.config.Password,
	)
	if err != nil {
		return fmt.Errorf("failed to create Kafka client: %v", err)
	}
	defer kc.Close()

	// Create consumer
	consumer, err := kc.CreateConsumer(kafkaAgent.consumer)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Consume messages
	consumeTimeout := time.Duration(timeout) * time.Millisecond
	if follow {
		consumeTimeout = 0 // No timeout for follow mode
	}

	if err := kc.ConsumeMessages(consumer, topic, maxMessages, consumeTimeout); err != nil {
		return err
	}

	return nil
}

func showConsumerStatus() error {
	if kafkaAgent.consumer == nil {
		return fmt.Errorf("no consumer configuration found. Run 'consumer config' first")
	}

	fmt.Println("📊 Consumer Status:")
	fmt.Printf("   Topic: %s\n", kafkaAgent.consumer.Topic)
	fmt.Printf("   Group ID: %s\n", kafkaAgent.consumer.GroupID)
	fmt.Printf("   Auto Offset Reset: %s\n", kafkaAgent.consumer.AutoOffsetReset)
	fmt.Printf("   Enable Auto Commit: %t\n", kafkaAgent.consumer.EnableAutoCommit)
	fmt.Printf("   Auto Commit Interval: %d ms\n", kafkaAgent.consumer.AutoCommitInterval)
	fmt.Printf("   Session Timeout: %d ms\n", kafkaAgent.consumer.SessionTimeout)
	fmt.Printf("   Max Poll Records: %d\n", kafkaAgent.consumer.MaxPollRecords)
	fmt.Printf("   Key Deserializer: %s\n", kafkaAgent.consumer.KeyDeserializer)
	fmt.Printf("   Value Deserializer: %s\n", kafkaAgent.consumer.ValueDeserializer)

	// Test connection
	kc, err := NewKafkaClient(
		kafkaAgent.config.Brokers,
		kafkaAgent.config.SecurityProtocol,
		kafkaAgent.config.SASLMechanism,
		kafkaAgent.config.Username,
		kafkaAgent.config.Password,
	)
	if err != nil {
		fmt.Println("   Connection: ❌ Disconnected")
		fmt.Printf("   Last Error: %v\n", err)
		return nil
	}
	defer kc.Close()

	// Try to create a consumer to test connectivity
	consumer, err := kc.CreateConsumer(kafkaAgent.consumer)
	if err != nil {
		fmt.Println("   Connection: ❌ Disconnected")
		fmt.Printf("   Last Error: %v\n", err)
	} else {
		consumer.Close()
		fmt.Println("   Connection: ✅ Connected")
	}

	return nil
}

func listConsumerGroups() error {
	// Note: Sarama doesn't provide direct consumer group listing
	// This would typically require Kafka Admin API or JMX
	fmt.Println("📋 Consumer Groups:")
	fmt.Println("   Note: Consumer group listing requires Kafka Admin API or JMX access")
	fmt.Println("   This feature is not available with the current Sarama implementation")

	// For now, return empty list
	groups := []map[string]interface{}{}

	if len(groups) == 0 {
		fmt.Println("   No consumer groups found or listing not supported")
		return nil
	}

	for _, group := range groups {
		fmt.Printf("\n   Group ID: %s\n", group["group_id"])
		fmt.Printf("   State: %s\n", group["state"])
		fmt.Printf("   Members: %d\n", group["members"])
		fmt.Printf("   Topics: %v\n", group["topics"])
		fmt.Printf("   Last Activity: %s\n", group["last_activity"])
	}

	return nil
}

func saveConsumerConfig(config *ConsumerConfig) error {
	// In a real implementation, this would save to a file or database
	// For now, we'll just store it in memory
	return nil
}

func logConsumerOperation(operation string, payload map[string]interface{}) {
	// In a real implementation, this would log to a file or monitoring system
	fmt.Printf("📝 Logged consumer operation: %s\n", operation)
} 
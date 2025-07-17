package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var topicCmd = &cobra.Command{
	Use:   "topic",
	Short: "Manage Kafka topics",
	Long:  `Create, delete, list, and configure Kafka topics.`,
}

var topicListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all topics",
	Long:  `List all topics in the Kafka cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := listTopics(); err != nil {
			fmt.Printf("❌ Error listing topics: %v\n", err)
			os.Exit(1)
		}
	},
}

var topicCreateCmd = &cobra.Command{
	Use:   "create [topic]",
	Short: "Create a new topic",
	Long:  `Create a new Kafka topic with specified configuration.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		partitions, _ := cmd.Flags().GetInt("partitions")
		replicationFactor, _ := cmd.Flags().GetInt("replication-factor")
		cleanupPolicy, _ := cmd.Flags().GetString("cleanup-policy")
		retentionMs, _ := cmd.Flags().GetInt("retention-ms")
		segmentBytes, _ := cmd.Flags().GetInt("segment-bytes")

		configs := map[string]*string{
			"cleanup.policy": ptrString(cleanupPolicy),
			"retention.ms":   ptrString(strconv.Itoa(retentionMs)),
			"segment.bytes":  ptrString(strconv.Itoa(segmentBytes)),
		}

		if err := createTopic(args[0], int32(partitions), int16(replicationFactor), configs); err != nil {
			fmt.Printf("❌ Error creating topic: %v\n", err)
			os.Exit(1)
		}
	},
}

var topicDeleteCmd = &cobra.Command{
	Use:   "delete [topic]",
	Short: "Delete a topic",
	Long:  `Delete a Kafka topic.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := deleteTopic(args[0]); err != nil {
			fmt.Printf("❌ Error deleting topic: %v\n", err)
			os.Exit(1)
		}
	},
}

var topicDescribeCmd = &cobra.Command{
	Use:   "describe [topic]",
	Short: "Describe a topic",
	Long:  `Show detailed information about a Kafka topic.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := describeTopic(args[0]); err != nil {
			fmt.Printf("❌ Error describing topic: %v\n", err)
			os.Exit(1)
		}
	},
}

var topicConfigCmd = &cobra.Command{
	Use:   "config [topic]",
	Short: "Show topic configuration",
	Long:  `Display configuration for a specific topic.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := showTopicConfig(args[0]); err != nil {
			fmt.Printf("❌ Error showing topic config: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Topic create flags
	topicCreateCmd.Flags().Int("partitions", 1, "Number of partitions")
	topicCreateCmd.Flags().Int("replication-factor", 1, "Replication factor")
	topicCreateCmd.Flags().String("cleanup-policy", "delete", "Cleanup policy (delete, compact)")
	topicCreateCmd.Flags().Int("retention-ms", 604800000, "Retention time in milliseconds (7 days)")
	topicCreateCmd.Flags().Int("segment-bytes", 1073741824, "Segment size in bytes (1GB)")

	topicCmd.AddCommand(topicListCmd, topicCreateCmd, topicDeleteCmd, topicDescribeCmd, topicConfigCmd)
	rootCmd.AddCommand(topicCmd)
}

func listTopics() error {
	kc, err := NewKafkaClient(
		kafkaAgent.config.Brokers,
		kafkaAgent.config.SecurityProtocol,
		kafkaAgent.config.SASLMechanism,
		kafkaAgent.config.Username,
		kafkaAgent.config.Password,
	)
	if err != nil {
		return err
	}
	defer kc.Close()

	fmt.Println("📋 Kafka Topics:")

	topics, err := kc.ListTopics()
	if err != nil {
		return err
	}

	if len(topics) == 0 {
		fmt.Println("   No topics found.")
		return nil
	}

	for _, topic := range topics {
		fmt.Printf("\n📄 Topic: %s\n", topic.Name)
		fmt.Printf("   Partitions: %d\n", topic.Partitions)
		fmt.Printf("   Replicas: %d\n", topic.Replicas)
		fmt.Printf("   Status: %s\n", "🟢 Active")
	}

	fmt.Printf("\n📊 Summary: %d topics found\n", len(topics))
	return nil
}

func createTopic(topicName string, partitions int32, replicationFactor int16, configs map[string]*string) error {
	kc, err := NewKafkaClient(
		kafkaAgent.config.Brokers,
		kafkaAgent.config.SecurityProtocol,
		kafkaAgent.config.SASLMechanism,
		kafkaAgent.config.Username,
		kafkaAgent.config.Password,
	)
	if err != nil {
		return err
	}
	defer kc.Close()

	fmt.Printf("🛠️  Creating topic: %s\n", topicName)
	fmt.Printf("   Partitions: %d\n", partitions)
	fmt.Printf("   Replication Factor: %d\n", replicationFactor)
	for k, v := range configs {
		fmt.Printf("   %s: %s\n", k, derefString(v))
	}

	if err := kc.CreateTopic(topicName, partitions, replicationFactor, configs); err != nil {
		return err
	}

	fmt.Println("✅ Topic created successfully!")
	return nil
}

func deleteTopic(topicName string) error {
	kc, err := NewKafkaClient(
		kafkaAgent.config.Brokers,
		kafkaAgent.config.SecurityProtocol,
		kafkaAgent.config.SASLMechanism,
		kafkaAgent.config.Username,
		kafkaAgent.config.Password,
	)
	if err != nil {
		return err
	}
	defer kc.Close()

	fmt.Printf("🗑️  Deleting topic: %s\n", topicName)
	if err := kc.DeleteTopic(topicName); err != nil {
		return err
	}
	fmt.Println("✅ Topic deleted successfully!")
	return nil
}

func describeTopic(topicName string) error {
	kc, err := NewKafkaClient(
		kafkaAgent.config.Brokers,
		kafkaAgent.config.SecurityProtocol,
		kafkaAgent.config.SASLMechanism,
		kafkaAgent.config.Username,
		kafkaAgent.config.Password,
	)
	if err != nil {
		return err
	}
	defer kc.Close()

	fmt.Printf("📄 Topic Details: %s\n", topicName)
	topic, err := kc.DescribeTopic(topicName)
	if err != nil {
		return err
	}

	fmt.Printf("   Name: %s\n", topic.Name)
	fmt.Printf("   Partitions: %d\n", topic.Partitions)
	fmt.Printf("   Replicas: %d\n", topic.Replicas)

	fmt.Printf("\n📊 Partition Information:\n")
	for _, partition := range topic.PartitionInfo {
		fmt.Printf("   Partition %d:\n", partition.Partition)
		fmt.Printf("     Leader: %d\n", partition.Leader)
		fmt.Printf("     Replicas: %v\n", partition.Replicas)
		fmt.Printf("     ISR: %v\n", partition.ISR)
		fmt.Printf("     Status: %s\n", partition.Status)
	}

	fmt.Printf("\n⚙️  Configuration:\n")
	for key, value := range topic.Configs {
		fmt.Printf("   %s: %s\n", key, value)
	}

	return nil
}

func showTopicConfig(topicName string) error {
	kc, err := NewKafkaClient(
		kafkaAgent.config.Brokers,
		kafkaAgent.config.SecurityProtocol,
		kafkaAgent.config.SASLMechanism,
		kafkaAgent.config.Username,
		kafkaAgent.config.Password,
	)
	if err != nil {
		return err
	}
	defer kc.Close()

	fmt.Printf("⚙️  Topic Configuration: %s\n", topicName)
	topic, err := kc.DescribeTopic(topicName)
	if err != nil {
		return err
	}

	keyConfigs := []string{
		"cleanup.policy", "retention.ms", "retention.bytes", "segment.bytes",
		"segment.ms", "min.cleanable.dirty.ratio", "delete.retention.ms",
		"min.compaction.lag.ms", "max.compaction.lag.ms", "min.insync.replicas",
		"compression.type", "message.format.version", "max.message.bytes",
	}

	for _, key := range keyConfigs {
		if value, exists := topic.Configs[key]; exists {
			fmt.Printf("   %s: %s\n", key, value)
		}
	}

	return nil
}

// Helper functions
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
} 
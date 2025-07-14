package main

import (
	"fmt"
	"strconv"
	"strings"

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
		config := make(map[string]interface{})
		config["partitions"], _ = cmd.Flags().GetInt("partitions")
		config["replication-factor"], _ = cmd.Flags().GetInt("replication-factor")
		config["cleanup-policy"], _ = cmd.Flags().GetString("cleanup-policy")
		config["retention-ms"], _ = cmd.Flags().GetInt("retention-ms")
		config["segment-bytes"], _ = cmd.Flags().GetInt("segment-bytes")
		
		if err := createTopic(args[0], config); err != nil {
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
	fmt.Println("�� Kafka Topics:")
	
	topics := getTopicList()
	
	if len(topics) == 0 {
		fmt.Println("   No topics found.")
		return nil
	}
	
	for _, topic := range topics {
		fmt.Printf("\n📄 Topic: %s\n", topic.Name)
		fmt.Printf("   Partitions: %d\n", topic.Partitions)
		fmt.Printf("   Replicas: %d\n", topic.Replicas)
		fmt.Printf("   Status: %s\n", getTopicStatus(topic))
	}
	
	fmt.Printf("\n�� Summary: %d topics found\n", len(topics))
	return nil
}

func createTopic(topicName string, config map[string]interface{}) error {
	fmt.Printf("�� Creating topic: %s\n", topicName)
	
	partitions := getInt(config, "partitions")
	replicationFactor := getInt(config, "replication-factor")
	cleanupPolicy := getString(config, "cleanup-policy")
	retentionMs := getInt(config, "retention-ms")
	segmentBytes := getInt(config, "segment-bytes")
	
	// Validate parameters
	if partitions <= 0 {
		return fmt.Errorf("partitions must be greater than 0")
	}
	if replicationFactor <= 0 {
		return fmt.Errorf("replication factor must be greater than 0")
	}
	
	// Simulate topic creation
	fmt.Printf("   Partitions: %d\n", partitions)
	fmt.Printf("   Replication Factor: %d\n", replicationFactor)
	fmt.Printf("   Cleanup Policy: %s\n", cleanupPolicy)
	fmt.Printf("   Retention: %d ms\n", retentionMs)
	fmt.Printf("   Segment Size: %d bytes\n", segmentBytes)
	
	// Here you would integrate with actual Kafka client
	// For now, we'll simulate success
	fmt.Println("✅ Topic created successfully!")
	
	// Log the operation
	logTopicOperation("create", map[string]interface{}{
		"topic":              topicName,
		"partitions":         partitions,
		"replication_factor": replicationFactor,
		"cleanup_policy":     cleanupPolicy,
		"retention_ms":       retentionMs,
		"segment_bytes":      segmentBytes,
		"timestamp":          getCurrentTimestamp(),
	})
	
	return nil
}

func deleteTopic(topicName string) error {
	fmt.Printf("🗑️  Deleting topic: %s\n", topicName)
	
	// Check if topic exists
	topics := getTopicList()
	topicExists := false
	for _, topic := range topics {
		if topic.Name == topicName {
			topicExists = true
			break
		}
	}
	
	if !topicExists {
		return fmt.Errorf("topic '%s' does not exist", topicName)
	}
	
	// Simulate topic deletion
	fmt.Printf("   Topic: %s\n", topicName)
	fmt.Printf("   Partitions: %d\n", getTopicPartitionCount(topicName))
	
	// Here you would integrate with actual Kafka client
	// For now, we'll simulate success
	fmt.Println("✅ Topic deleted successfully!")
	
	// Log the operation
	logTopicOperation("delete", map[string]interface{}{
		"topic":     topicName,
		"timestamp": getCurrentTimestamp(),
	})
	
	return nil
}

func describeTopic(topicName string) error {
	fmt.Printf("📄 Topic Details: %s\n", topicName)
	
	// Get topic information
	topic := getTopicInfo(topicName)
	if topic == nil {
		return fmt.Errorf("topic '%s' not found", topicName)
	}
	
	fmt.Printf("   Name: %s\n", topic.Name)
	fmt.Printf("   Partitions: %d\n", topic.Partitions)
	fmt.Printf("   Replicas: %d\n", topic.Replicas)
	
	// Show partition details
	fmt.Printf("\n📊 Partition Information:\n")
	for _, partition := range topic.PartitionInfo {
		fmt.Printf("   Partition %d:\n", partition.Partition)
		fmt.Printf("     Leader: %d\n", partition.Leader)
		fmt.Printf("     Replicas: %v\n", partition.Replicas)
		fmt.Printf("     ISR: %v\n", partition.ISR)
		fmt.Printf("     Status: %s\n", partition.Status)
	}
	
	// Show configuration
	fmt.Printf("\n⚙️  Configuration:\n")
	for key, value := range topic.Configs {
		fmt.Printf("   %s: %s\n", key, value)
	}
	
	return nil
}

func showTopicConfig(topicName string) error {
	fmt.Printf("⚙️  Topic Configuration: %s\n", topicName)
	
	// Get topic configuration
	config := getTopicConfig(topicName)
	if config == nil {
		return fmt.Errorf("topic '%s' not found", topicName)
	}
	
	// Show configuration parameters
	keyConfigs := []string{
		"cleanup.policy", "retention.ms", "retention.bytes", "segment.bytes",
		"segment.ms", "min.cleanable.dirty.ratio", "delete.retention.ms",
		"min.compaction.lag.ms", "max.compaction.lag.ms", "min.insync.replicas",
		"compression.type", "message.format.version", "max.message.bytes",
	}
	
	for _, key := range keyConfigs {
		if value, exists := config[key]; exists {
			fmt.Printf("   %s: %s\n", key, value)
		}
	}
	
	return nil
}

// Helper functions

func getTopicList() []TopicInfo {
	// Simulate topic list
	return []TopicInfo{
		{
			Name:       "test-topic-1",
			Partitions: 3,
			Replicas:   2,
			Configs: map[string]string{
				"cleanup.policy": "delete",
				"retention.ms":   "604800000",
			},
		},
		{
			Name:       "test-topic-2",
			Partitions: 1,
			Replicas:   1,
			Configs: map[string]string{
				"cleanup.policy": "compact",
				"retention.ms":   "86400000",
			},
		},
		{
			Name:       "events",
			Partitions: 5,
			Replicas:   3,
			Configs: map[string]string{
				"cleanup.policy": "delete",
				"retention.ms":   "2592000000",
			},
		},
	}
}

func getTopicInfo(topicName string) *TopicInfo {
	topics := getTopicList()
	for _, topic := range topics {
		if topic.Name == topicName {
			// Add partition information
			topic.PartitionInfo = []PartitionInfo{
				{
					Partition: 0,
					Leader:    0,
					Replicas:  []int{0, 1},
					ISR:       []int{0, 1},
					Status:    "Online",
				},
				{
					Partition: 1,
					Leader:    1,
					Replicas:  []int{1, 2},
					ISR:       []int{1, 2},
					Status:    "Online",
				},
			}
			return &topic
		}
	}
	return nil
}

func getTopicConfig(topicName string) map[string]string {
	// Simulate topic configuration
	return map[string]string{
		"cleanup.policy":           "delete",
		"retention.ms":             "604800000",
		"retention.bytes":          "-1",
		"segment.bytes":            "1073741824",
		"segment.ms":               "604800000",
		"min.cleanable.dirty.ratio": "0.5",
		"delete.retention.ms":      "86400000",
		"min.compaction.lag.ms":    "0",
		"max.compaction.lag.ms":    "9223372036854775807",
		"min.insync.replicas":      "1",
		"compression.type":         "producer",
		"message.format.version":   "2.8-IV1",
		"max.message.bytes":        "1048588",
	}
}

func getTopicStatus(topic TopicInfo) string {
	// Simulate topic status
	return "🟢 Active"
}

func getTopicPartitionCount(topicName string) int {
	topic := getTopicInfo(topicName)
	if topic != nil {
		return topic.Partitions
	}
	return 0
}

func logTopicOperation(operation string, payload map[string]interface{}) {
	// In a real implementation, this would log to a file or monitoring system
	fmt.Printf("📝 Logged topic operation: %s\n", operation)
} 
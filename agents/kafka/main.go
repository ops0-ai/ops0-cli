package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// KafkaConfig represents the configuration for Kafka connections
type KafkaConfig struct {
	Brokers          []string `json:"brokers" yaml:"brokers"`
	SecurityProtocol string   `json:"security_protocol" yaml:"security_protocol"`
	SASLMechanism    string   `json:"sasl_mechanism" yaml:"sasl_mechanism"`
	Username         string   `json:"username" yaml:"username"`
	Password         string   `json:"password" yaml:"password"`
	SSLCAFile        string   `json:"ssl_ca_file" yaml:"ssl_ca_file"`
	SSLCertFile      string   `json:"ssl_cert_file" yaml:"ssl_cert_file"`
	SSLKeyFile       string   `json:"ssl_key_file" yaml:"ssl_key_file"`
}

// ProducerConfig represents Kafka producer configuration
type ProducerConfig struct {
	Topic           string            `json:"topic" yaml:"topic"`
	Acks            string            `json:"acks" yaml:"acks"`
	Retries         int               `json:"retries" yaml:"retries"`
	BatchSize       int               `json:"batch_size" yaml:"batch_size"`
	LingerMs        int               `json:"linger_ms" yaml:"linger_ms"`
	CompressionType string            `json:"compression_type" yaml:"compression_type"`
	KeySerializer   string            `json:"key_serializer" yaml:"key_serializer"`
	ValueSerializer string            `json:"value_serializer" yaml:"value_serializer"`
	Properties      map[string]string `json:"properties" yaml:"properties"`
}

// ConsumerConfig represents Kafka consumer configuration
type ConsumerConfig struct {
	Topic              string            `json:"topic" yaml:"topic"`
	GroupID            string            `json:"group_id" yaml:"group_id"`
	AutoOffsetReset    string            `json:"auto_offset_reset" yaml:"auto_offset_reset"`
	EnableAutoCommit   bool              `json:"enable_auto_commit" yaml:"enable_auto_commit"`
	AutoCommitInterval int               `json:"auto_commit_interval" yaml:"auto_commit_interval"`
	SessionTimeout     int               `json:"session_timeout" yaml:"session_timeout"`
	MaxPollRecords     int               `json:"max_poll_records" yaml:"max_poll_records"`
	KeyDeserializer    string            `json:"key_deserializer" yaml:"key_deserializer"`
	ValueDeserializer  string            `json:"value_deserializer" yaml:"value_deserializer"`
	Properties         map[string]string `json:"properties" yaml:"properties"`
}

// BrokerInfo represents information about a Kafka broker
type BrokerInfo struct {
	ID       int    `json:"id"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Rack     string `json:"rack"`
	Status   string `json:"status"`
	Leader   bool   `json:"leader"`
	Replicas int    `json:"replicas"`
}

// TopicInfo represents information about a Kafka topic
type TopicInfo struct {
	Name          string                   `json:"name"`
	Partitions    int                      `json:"partitions"`
	Replicas      int                      `json:"replicas"`
	Configs       map[string]string        `json:"configs"`
	PartitionInfo []PartitionInfo          `json:"partition_info"`
}

// PartitionInfo represents information about a topic partition
type PartitionInfo struct {
	Partition int    `json:"partition"`
	Leader    int    `json:"leader"`
	Replicas  []int  `json:"replicas"`
	ISR       []int  `json:"isr"`
	Status    string `json:"status"`
}

// KafkaAgent represents the main Kafka agent
type KafkaAgent struct {
	config     *KafkaConfig
	producer   *ProducerConfig
	consumer   *ConsumerConfig
	connection *KafkaConnection
}

// KafkaConnection represents the connection to Kafka cluster
type KafkaConnection struct {
	Brokers   []string
	Connected bool
	LastError error
	Metadata  map[string]interface{}
}

// KafkaCommand represents a Kafka operation command
type KafkaCommand struct {
	Operation string                 `json:"operation"`
	Topic     string                 `json:"topic,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Key       string                 `json:"key,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

var (
	configFile string
	kafkaAgent *KafkaAgent
	rootCmd    = &cobra.Command{
		Use:   "kafka-agent",
		Short: "A comprehensive Kafka management agent",
		Long: `Kafka Agent - A comprehensive tool for managing Kafka producers, consumers, and brokers.
		
This agent provides:
- Producer configuration and management
- Consumer configuration and management  
- Broker status and health monitoring
- Topic management and monitoring
- Cluster configuration management`,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	
	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.kafka-agent.yaml)")
	rootCmd.PersistentFlags().StringSlice("brokers", []string{"localhost:9092"}, "Kafka broker addresses")
	rootCmd.PersistentFlags().String("security-protocol", "PLAINTEXT", "Security protocol (PLAINTEXT, SSL, SASL_PLAINTEXT, SASL_SSL)")
	rootCmd.PersistentFlags().String("sasl-mechanism", "", "SASL mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)")
	rootCmd.PersistentFlags().String("username", "", "SASL username")
	rootCmd.PersistentFlags().String("password", "", "SASL password")
	
	// Bind flags to viper
	viper.BindPFlag("brokers", rootCmd.PersistentFlags().Lookup("brokers"))
	viper.BindPFlag("security_protocol", rootCmd.PersistentFlags().Lookup("security-protocol"))
	viper.BindPFlag("sasl_mechanism", rootCmd.PersistentFlags().Lookup("sasl-mechanism"))
	viper.BindPFlag("username", rootCmd.PersistentFlags().Lookup("username"))
	viper.BindPFlag("password", rootCmd.PersistentFlags().Lookup("password"))
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kafka-agent")
	}
	
	viper.AutomaticEnv()
	
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
	
	// Initialize Kafka agent
	kafkaAgent = &KafkaAgent{
		config: &KafkaConfig{
			Brokers:          viper.GetStringSlice("brokers"),
			SecurityProtocol: viper.GetString("security_protocol"),
			SASLMechanism:    viper.GetString("sasl_mechanism"),
			Username:         viper.GetString("username"),
			Password:         viper.GetString("password"),
		},
		connection: &KafkaConnection{
			Brokers:   viper.GetStringSlice("brokers"),
			Connected: false,
		},
	}
}

// ExecuteKafkaCommand executes a Kafka operation
func ExecuteKafkaCommand(command *KafkaCommand) error {
	switch command.Operation {
	case "producer_config":
		return configureProducer(command.Config)
	case "producer_publish":
		return publishMessage(command.Topic, command.Message, command.Key, command.Metadata)
	case "consumer_config":
		return configureConsumer(command.Config)
	case "consumer_subscribe":
		return subscribeToTopic(command.Topic)
	case "consumer_consume":
		return consumeMessages(command.Topic, command.Metadata)
	case "broker_status":
		return showBrokerStatus()
	case "broker_health":
		return checkBrokerHealth()
	case "broker_config":
		return showBrokerConfig()
	case "broker_metrics":
		return showBrokerMetrics()
	case "broker_connect":
		return testBrokerConnectivity()
	case "topic_list":
		return listTopics()
	case "topic_create":
		// Extract topic creation parameters from config
		partitions := int32(getInt(command.Config, "partitions"))
		replicationFactor := int16(getInt(command.Config, "replication-factor"))
		cleanupPolicy := getString(command.Config, "cleanup-policy")
		retentionMs := getString(command.Config, "retention-ms")
		segmentBytes := getString(command.Config, "segment-bytes")
		
		configs := map[string]*string{
			"cleanup.policy": ptrString(cleanupPolicy),
			"retention.ms":   ptrString(retentionMs),
			"segment.bytes":  ptrString(segmentBytes),
		}
		return createTopic(command.Topic, partitions, replicationFactor, configs)
	case "topic_delete":
		return deleteTopic(command.Topic)
	case "topic_describe":
		return describeTopic(command.Topic)
	default:
		return fmt.Errorf("unknown operation: %s", command.Operation)
	}
}

// GetKafkaCommandSuggestion returns a Kafka command suggestion based on natural language input
func GetKafkaCommandSuggestion(userInput string) *CommandSuggestion {
	// Parse natural language input and return appropriate Kafka command
	// This integrates with your existing AI suggestion system
	
	// Example mappings (in a real implementation, this would use AI)
	userInput = strings.ToLower(userInput)
	
	var suggestion *CommandSuggestion
	
	switch {
	case strings.Contains(userInput, "produce") || strings.Contains(userInput, "publish"):
		suggestion = &CommandSuggestion{
			Tool:        "kafka",
			Command:     "kafka-agent producer publish",
			Description: "Publish a message to Kafka topic",
			Intent:      "producer_publish",
			Confidence:  0.9,
		}
	case strings.Contains(userInput, "consume") || strings.Contains(userInput, "read"):
		suggestion = &CommandSuggestion{
			Tool:        "kafka",
			Command:     "kafka-agent consumer consume",
			Description: "Consume messages from Kafka topic",
			Intent:      "consumer_consume",
			Confidence:  0.9,
		}
	case strings.Contains(userInput, "broker") && (strings.Contains(userInput, "status") || strings.Contains(userInput, "health")):
		suggestion = &CommandSuggestion{
			Tool:        "kafka",
			Command:     "kafka-agent broker status",
			Description: "Show Kafka broker status",
			Intent:      "broker_status",
			Confidence:  0.9,
		}
	case strings.Contains(userInput, "topic") && strings.Contains(userInput, "list"):
		suggestion = &CommandSuggestion{
			Tool:        "kafka",
			Command:     "kafka-agent topic list",
			Description: "List Kafka topics",
			Intent:      "topic_list",
			Confidence:  0.9,
		}
	default:
		suggestion = &CommandSuggestion{
			Tool:        "kafka",
			Command:     "kafka-agent --help",
			Description: "Show Kafka agent help",
			Intent:      "help",
			Confidence:  0.5,
		}
	}
	
	return suggestion
}

// Helper functions
func ptrString(s string) *string {
	return &s
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

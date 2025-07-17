package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
)

type KafkaClient struct {
	config  *sarama.Config
	client  sarama.Client
	admin   sarama.ClusterAdmin
	brokers []string
}

// NewKafkaClient creates a new Kafka client and admin connection.
func NewKafkaClient(brokers []string, securityProtocol, saslMechanism, username, password string) (*KafkaClient, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0 // Adjust if your Kafka version is different

	// Security configuration
	switch securityProtocol {
	case "SSL":
		config.Net.TLS.Enable = true
	case "SASL_PLAINTEXT", "SASL_SSL":
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLMechanism(saslMechanism)
		config.Net.SASL.User = username
		config.Net.SASL.Password = password
		if securityProtocol == "SASL_SSL" {
			config.Net.TLS.Enable = true
		}
	}

	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %v", err)
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create admin client: %v", err)
	}

	return &KafkaClient{
		config:  config,
		client:  client,
		admin:   admin,
		brokers: brokers,
	}, nil
}

func (kc *KafkaClient) Close() {
	if kc.admin != nil {
		kc.admin.Close()
	}
	if kc.client != nil {
		kc.client.Close()
	}
}

// Topic Operations
func (kc *KafkaClient) ListTopics() ([]TopicInfo, error) {
	topics, err := kc.admin.ListTopics()
	if err != nil {
		return nil, fmt.Errorf("failed to list topics: %v", err)
	}

	var topicInfos []TopicInfo
	for name, detail := range topics {
		topicInfo := TopicInfo{
			Name:       name,
			Partitions: int(detail.NumPartitions),
			Replicas:   int(detail.ReplicationFactor),
			Configs:    make(map[string]string),
		}

		// Get topic configs
		configs, err := kc.admin.DescribeConfig(sarama.ConfigResource{
			Type: sarama.TopicResource,
			Name: name,
		})
		if err == nil {
			for _, config := range configs {
				topicInfo.Configs[config.Name] = config.Value
			}
		}

		topicInfos = append(topicInfos, topicInfo)
	}

	return topicInfos, nil
}

func (kc *KafkaClient) CreateTopic(name string, partitions int32, replicationFactor int16, configs map[string]*string) error {
	return kc.admin.CreateTopic(name, &sarama.TopicDetail{
		NumPartitions:     partitions,
		ReplicationFactor: replicationFactor,
		ConfigEntries:     configs,
	}, false)
}

func (kc *KafkaClient) DeleteTopic(name string) error {
	return kc.admin.DeleteTopic(name)
}

func (kc *KafkaClient) DescribeTopic(name string) (*TopicInfo, error) {
	topics, err := kc.admin.ListTopics()
	if err != nil {
		return nil, err
	}

	detail, exists := topics[name]
	if !exists {
		return nil, fmt.Errorf("topic '%s' not found", name)
	}

	topicInfo := &TopicInfo{
		Name:       name,
		Partitions: int(detail.NumPartitions),
		Replicas:   int(detail.ReplicationFactor),
		Configs:    make(map[string]string),
	}

	// Get partition info
	partitions, err := kc.client.Partitions(name)
	if err == nil {
		for _, partition := range partitions {
			leader, err := kc.client.Leader(name, partition)
			if err != nil {
				continue
			}
			replicas, err := kc.client.Replicas(name, partition)
			if err != nil {
				continue
			}
			isr, err := kc.client.InSyncReplicas(name, partition)
			if err != nil {
				continue
			}
			partitionInfo := PartitionInfo{
				Partition: int(partition),
				Leader:    int(leader.ID()),
				Replicas:  make([]int, len(replicas)),
				ISR:       make([]int, len(isr)),
				Status:    "Online",
			}
			for i, replica := range replicas {
				partitionInfo.Replicas[i] = int(replica)
			}
			for i, isrReplica := range isr {
				partitionInfo.ISR[i] = int(isrReplica)
			}
			topicInfo.PartitionInfo = append(topicInfo.PartitionInfo, partitionInfo)
		}
	}

	// Get topic configs
	configs, err := kc.admin.DescribeConfig(sarama.ConfigResource{
		Type: sarama.TopicResource,
		Name: name,
	})
	if err == nil {
		for _, config := range configs {
			topicInfo.Configs[config.Name] = config.Value
		}
	}

	return topicInfo, nil
}

// Broker Operations
func (kc *KafkaClient) GetBrokerInfo() ([]BrokerInfo, error) {
	brokers := kc.client.Brokers()
	var brokerInfos []BrokerInfo

	for _, broker := range brokers {
		brokerInfo := BrokerInfo{
			ID:     int(broker.ID()),
			Host:   broker.Addr(),
			Port:   9092, // Default port, you might want to parse this from broker.Addr()
			Rack:   "",
			Status: "Online",
			Leader: false,
		}

		// Try to get broker metadata
		if connected, err := broker.Connected(); err == nil && connected {
			brokerInfo.Status = "Online"
		} else {
			brokerInfo.Status = "Offline"
		}

		brokerInfos = append(brokerInfos, brokerInfo)
	}

	return brokerInfos, nil
}

func (kc *KafkaClient) GetBrokerConfig(brokerID int) (map[string]string, error) {
	// Note: Getting broker configs via Sarama is limited
	// This would typically require JMX or REST API access
	configs := make(map[string]string)
	
	// Try to get some basic broker info
	brokers := kc.client.Brokers()
	for _, broker := range brokers {
		if int(broker.ID()) == brokerID {
			if connected, _ := broker.Connected(); connected {
				configs["broker.id"] = strconv.Itoa(brokerID)
				configs["listeners"] = "PLAINTEXT://:9092"
				configs["log.dirs"] = "/tmp/kafka-logs"
				// Add more configs as available
			}
			break
		}
	}

	return configs, nil
}

func (kc *KafkaClient) TestBrokerConnectivity() error {
	for _, broker := range kc.brokers {
		fmt.Printf("🖥️  Testing connection to %s...\n", broker)
		
		// Test TCP connection
		conn, err := kc.client.Brokers()[0].Connected()
		if err != nil {
			fmt.Printf("   ❌ TCP connection failed: %v\n", err)
			continue
		}
		fmt.Printf("   ✅ TCP connection successful\n")
		
		// Test Kafka protocol
		if conn {
			fmt.Printf("   ✅ Kafka protocol test successful\n")
		} else {
			fmt.Printf("   ❌ Kafka protocol test failed\n")
		}
	}
	
	return nil
}

// Producer Operations
func (kc *KafkaClient) CreateProducer(config *ProducerConfig) (sarama.SyncProducer, error) {
	producerConfig := sarama.NewConfig()
	producerConfig.Version = sarama.V2_8_0_0

	// Set acks
	switch config.Acks {
	case "0":
		producerConfig.Producer.RequiredAcks = sarama.NoResponse
	case "1":
		producerConfig.Producer.RequiredAcks = sarama.WaitForLocal
	case "all":
		producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	}

	// Set retries
	producerConfig.Producer.Retry.Max = config.Retries

	// Set batch size
	producerConfig.Producer.Flush.Bytes = config.BatchSize

	// Set linger
	producerConfig.Producer.Flush.Frequency = time.Duration(config.LingerMs) * time.Millisecond

	// Set compression
	switch config.CompressionType {
	case "gzip":
		producerConfig.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		producerConfig.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		producerConfig.Producer.Compression = sarama.CompressionLZ4
	}

	// Security configuration
	if kafkaAgent.config.SecurityProtocol != "PLAINTEXT" {
		producerConfig.Net.SASL.Enable = true
		producerConfig.Net.SASL.Mechanism = sarama.SASLMechanism(kafkaAgent.config.SASLMechanism)
		producerConfig.Net.SASL.User = kafkaAgent.config.Username
		producerConfig.Net.SASL.Password = kafkaAgent.config.Password
	}

	return sarama.NewSyncProducer(kc.brokers, producerConfig)
}

func (kc *KafkaClient) PublishMessage(producer sarama.SyncProducer, topic, message, key string, partition int32, headers map[string]string) error {
	msg := &sarama.ProducerMessage{
		Topic:     topic,
		Value:     sarama.StringEncoder(message),
		Partition: partition,
	}

	if key != "" {
		msg.Key = sarama.StringEncoder(key)
	}

	if len(headers) > 0 {
		msg.Headers = make([]sarama.RecordHeader, 0, len(headers))
		for k, v := range headers {
			msg.Headers = append(msg.Headers, sarama.RecordHeader{
				Key:   []byte(k),
				Value: []byte(v),
			})
		}
	}

	partition, offset, err := producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	fmt.Printf("✅ Message sent to partition %d at offset %d\n", partition, offset)
	return nil
}

// Consumer Operations
func (kc *KafkaClient) CreateConsumer(config *ConsumerConfig) (sarama.Consumer, error) {
	consumerConfig := sarama.NewConfig()
	consumerConfig.Version = sarama.V2_8_0_0

	// Set auto offset reset
	switch config.AutoOffsetReset {
	case "earliest":
		consumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	case "latest":
		consumerConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	// Set auto commit
	consumerConfig.Consumer.Offsets.AutoCommit.Enable = config.EnableAutoCommit
	consumerConfig.Consumer.Offsets.AutoCommit.Interval = time.Duration(config.AutoCommitInterval) * time.Millisecond

	// Set session timeout
	consumerConfig.Consumer.Group.Session.Timeout = time.Duration(config.SessionTimeout) * time.Millisecond

	// Set max poll records
	consumerConfig.Consumer.Fetch.Max = int32(config.MaxPollRecords)

	// Security configuration
	if kafkaAgent.config.SecurityProtocol != "PLAINTEXT" {
		consumerConfig.Net.SASL.Enable = true
		consumerConfig.Net.SASL.Mechanism = sarama.SASLMechanism(kafkaAgent.config.SASLMechanism)
		consumerConfig.Net.SASL.User = kafkaAgent.config.Username
		consumerConfig.Net.SASL.Password = kafkaAgent.config.Password
	}

	return sarama.NewConsumer(kc.brokers, consumerConfig)
}

func (kc *KafkaClient) ConsumeMessages(consumer sarama.Consumer, topic string, maxMessages int, timeout time.Duration) error {
	partitions, err := consumer.Partitions(topic)
	if err != nil {
		return fmt.Errorf("failed to get partitions: %v", err)
	}

	messageCount := 0
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for _, partition := range partitions {
		partitionConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetOldest)
		if err != nil {
			fmt.Printf("❌ Failed to create partition consumer for partition %d: %v\n", partition, err)
			continue
		}
		defer partitionConsumer.Close()

		for {
			select {
			case msg := <-partitionConsumer.Messages():
				fmt.Printf("\n📨 Message #%d:\n", messageCount+1)
				fmt.Printf("   Topic: %s\n", msg.Topic)
				fmt.Printf("   Partition: %d\n", msg.Partition)
				fmt.Printf("   Offset: %d\n", msg.Offset)
				if msg.Key != nil {
					fmt.Printf("   Key: %s\n", string(msg.Key))
				}
				fmt.Printf("   Value: %s\n", string(msg.Value))
				fmt.Printf("   Timestamp: %s\n", msg.Timestamp.Format(time.RFC3339))

				if len(msg.Headers) > 0 {
					fmt.Printf("   Headers: ")
					for _, header := range msg.Headers {
						fmt.Printf("%s=%s ", string(header.Key), string(header.Value))
					}
					fmt.Println()
				}

				messageCount++
				if maxMessages > 0 && messageCount >= maxMessages {
					return nil
				}

			case err := <-partitionConsumer.Errors():
				fmt.Printf("❌ Consumer error: %v\n", err)

			case <-ctx.Done():
				return nil
			}
		}
	}

	return nil
}

func (kc *KafkaClient) ListConsumerGroups() ([]map[string]interface{}, error) {
	// Note: Sarama doesn't provide direct consumer group listing
	// This would typically require Kafka Admin API or JMX
	// For now, return empty list
	return []map[string]interface{}{}, nil
}

// Helper function to parse broker address
func parseBrokerAddress(broker string) (string, int, error) {
	parts := strings.Split(broker, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid broker format: %s", broker)
	}

	host := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %s", parts[1])
	}

	return host, port, nil
} 
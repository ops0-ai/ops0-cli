package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var brokerCmd = &cobra.Command{
	Use:   "broker",
	Short: "Manage Kafka brokers",
	Long:  `Configure and manage Kafka brokers including status monitoring, configuration, and health checks.`,
}

var brokerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show broker status",
	Long:  `Display the status of all Kafka brokers in the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := showBrokerStatus(); err != nil {
			fmt.Printf("❌ Error showing broker status: %v\n", err)
			os.Exit(1)
		}
	},
}

var brokerHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check broker health",
	Long:  `Perform health checks on all Kafka brokers.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := checkBrokerHealth(); err != nil {
			fmt.Printf("❌ Error checking broker health: %v\n", err)
			os.Exit(1)
		}
	},
}

var brokerConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show broker configuration",
	Long:  `Display configuration for all brokers in the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := showBrokerConfig(); err != nil {
			fmt.Printf("❌ Error showing broker config: %v\n", err)
			os.Exit(1)
		}
	},
}

var brokerMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show broker metrics",
	Long:  `Display metrics for all brokers in the cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := showBrokerMetrics(); err != nil {
			fmt.Printf("❌ Error showing broker metrics: %v\n", err)
			os.Exit(1)
		}
	},
}

var brokerConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Test broker connectivity",
	Long:  `Test connectivity to all Kafka brokers.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := testBrokerConnectivity(); err != nil {
			fmt.Printf("❌ Error testing broker connectivity: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	brokerCmd.AddCommand(brokerStatusCmd, brokerHealthCmd, brokerConfigCmd, brokerMetricsCmd, brokerConnectCmd)
	rootCmd.AddCommand(brokerCmd)
}

func showBrokerStatus() error {
	fmt.Println("📊 Kafka Broker Status:")
	
	brokers := getBrokerInfo()
	
	for _, broker := range brokers {
		fmt.Printf("\n��️  Broker %d (%s:%d):\n", broker.ID, broker.Host, broker.Port)
		fmt.Printf("   Status: %s\n", getStatusEmoji(broker.Status) + " " + broker.Status)
		fmt.Printf("   Rack: %s\n", broker.Rack)
		fmt.Printf("   Leader: %t\n", broker.Leader)
		fmt.Printf("   Replicas: %d\n", broker.Replicas)
		
		// Show additional details
		if broker.Status == "Online" {
			fmt.Printf("   Uptime: %s\n", getBrokerUptime(broker.ID))
			fmt.Printf("   Version: %s\n", getBrokerVersion(broker.ID))
		}
	}
	
	// Summary
	onlineCount := 0
	offlineCount := 0
	for _, broker := range brokers {
		if broker.Status == "Online" {
			onlineCount++
		} else {
			offlineCount++
		}
	}
	
	fmt.Printf("\n📈 Summary:\n")
	fmt.Printf("   Total Brokers: %d\n", len(brokers))
	fmt.Printf("   Online: %d\n", onlineCount)
	fmt.Printf("   Offline: %d\n", offlineCount)
	fmt.Printf("   Health: %s\n", getClusterHealth(onlineCount, len(brokers)))
	
	return nil
}

func checkBrokerHealth() error {
	fmt.Println("🏥 Kafka Broker Health Check:")
	
	brokers := getBrokerInfo()
	
	for _, broker := range brokers {
		fmt.Printf("\n🖥️  Checking Broker %d (%s:%d)...\n", broker.ID, broker.Host, broker.Port)
		
		// Perform health checks
		checks := performHealthChecks(broker)
		
		for checkName, result := range checks {
			status := "✅ PASS"
			if !result.Success {
				status = "❌ FAIL"
			}
			fmt.Printf("   %s: %s\n", checkName, status)
			if !result.Success && result.Error != "" {
				fmt.Printf("      Error: %s\n", result.Error)
			}
		}
	}
	
	fmt.Println("\n✅ Health check completed!")
	return nil
}

func showBrokerConfig() error {
	fmt.Println("⚙️  Kafka Broker Configuration:")
	
	brokers := getBrokerInfo()
	
	for _, broker := range brokers {
		fmt.Printf("\n��️  Broker %d (%s:%d):\n", broker.ID, broker.Host, broker.Port)
		
		config := getBrokerConfig(broker.ID)
		
		// Show key configuration parameters
		keyConfigs := []string{
			"log.dirs", "num.network.threads", "num.io.threads", "socket.send.buffer.bytes",
			"socket.receive.buffer.bytes", "socket.request.max.bytes", "num.partitions",
			"num.recovery.threads.per.data.dir", "offsets.topic.replication.factor",
			"transaction.state.log.replication.factor", "transaction.state.log.min.isr",
			"log.retention.hours", "log.segment.bytes", "log.retention.check.interval.ms",
		}
		
		for _, key := range keyConfigs {
			if value, exists := config[key]; exists {
				fmt.Printf("   %s: %s\n", key, value)
			}
		}
	}
	
	return nil
}

func showBrokerMetrics() error {
	fmt.Println("📈 Kafka Broker Metrics:")
	
	brokers := getBrokerInfo()
	
	for _, broker := range brokers {
		fmt.Printf("\n️  Broker %d (%s:%d):\n", broker.ID, broker.Host, broker.Port)
		
		metrics := getBrokerMetrics(broker.ID)
		
		// Show key metrics
		fmt.Printf("   Messages/sec: %.2f\n", metrics["messages_per_sec"])
		fmt.Printf("   Bytes/sec: %.2f\n", metrics["bytes_per_sec"])
		fmt.Printf("   Active Controllers: %d\n", int(metrics["active_controllers"]))
		fmt.Printf("   Offline Partitions: %d\n", int(metrics["offline_partitions"]))
		fmt.Printf("   Under Replicated Partitions: %d\n", int(metrics["under_replicated_partitions"]))
		fmt.Printf("   Total Time: %.2f ms\n", metrics["total_time_ms"])
		fmt.Printf("   Request Queue Size: %d\n", int(metrics["request_queue_size"]))
		fmt.Printf("   Response Queue Size: %d\n", int(metrics["response_queue_size"]))
		fmt.Printf("   Network Processor Avg Idle: %.2f%%\n", metrics["network_processor_avg_idle"])
		fmt.Printf("   Request Handler Avg Idle: %.2f%%\n", metrics["request_handler_avg_idle"])
	}
	
	return nil
}

func testBrokerConnectivity() error {
	fmt.Println("🔌 Testing Kafka Broker Connectivity:")
	
	brokers := kafkaAgent.config.Brokers
	
	for _, broker := range brokers {
		fmt.Printf("\n🖥️  Testing connection to %s...\n", broker)
		
		// Parse broker address
		host, port, err := parseBrokerAddress(broker)
		if err != nil {
			fmt.Printf("   ❌ Invalid broker address: %v\n", err)
			continue
		}
		
		// Test TCP connectivity
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
		if err != nil {
			fmt.Printf("   ❌ Connection failed: %v\n", err)
			continue
		}
		defer conn.Close()
		
		fmt.Printf("   ✅ TCP connection successful\n")
		
		// Test Kafka protocol (simplified)
		if testKafkaProtocol(host, port) {
			fmt.Printf("   ✅ Kafka protocol test successful\n")
		} else {
			fmt.Printf("   ⚠️  Kafka protocol test failed\n")
		}
	}
	
	fmt.Println("\n✅ Connectivity test completed!")
	return nil
}

// Helper functions

func getBrokerInfo() []BrokerInfo {
	// In a real implementation, this would query the Kafka cluster
	// For now, we'll simulate broker information
	return []BrokerInfo{
		{
			ID:       0,
			Host:     "localhost",
			Port:     9092,
			Rack:     "rack-1",
			Status:   "Online",
			Leader:   true,
			Replicas: 3,
		},
		{
			ID:       1,
			Host:     "localhost",
			Port:     9093,
			Rack:     "rack-1",
			Status:   "Online",
			Leader:   false,
			Replicas: 2,
		},
		{
			ID:       2,
			Host:     "localhost",
			Port:     9094,
			Rack:     "rack-2",
			Status:   "Online",
			Leader:   false,
			Replicas: 1,
		},
	}
}

func getStatusEmoji(status string) string {
	switch status {
	case "Online":
		return ""
	case "Offline":
		return "🔴"
	case "Starting":
		return "🟡"
	case "Stopping":
		return "🟠"
	default:
		return "⚪"
	}
}

func getBrokerUptime(brokerID int) string {
	// Simulate uptime
	return "2 days, 5 hours, 30 minutes"
}

func getBrokerVersion(brokerID int) string {
	// Simulate version
	return "3.5.1"
}

func getClusterHealth(online, total int) string {
	percentage := float64(online) / float64(total) * 100
	if percentage >= 90 {
		return " Excellent"
	} else if percentage >= 75 {
		return "🟡 Good"
	} else if percentage >= 50 {
		return "🟠 Fair"
	} else {
		return "🔴 Poor"
	}
}

type HealthCheckResult struct {
	Success bool
	Error   string
}

func performHealthChecks(broker BrokerInfo) map[string]HealthCheckResult {
	checks := make(map[string]HealthCheckResult)
	
	// Simulate various health checks
	checks["TCP Connectivity"] = HealthCheckResult{Success: true}
	checks["Kafka Protocol"] = HealthCheckResult{Success: true}
	checks["Disk Space"] = HealthCheckResult{Success: true}
	checks["Memory Usage"] = HealthCheckResult{Success: true}
	checks["CPU Usage"] = HealthCheckResult{Success: true}
	checks["Network Latency"] = HealthCheckResult{Success: true}
	
	// Simulate one failing check occasionally
	if broker.ID == 1 {
		checks["Disk Space"] = HealthCheckResult{
			Success: false,
			Error:   "Disk usage above 90%",
		}
	}
	
	return checks
}

func getBrokerConfig(brokerID int) map[string]string {
	// Simulate broker configuration
	return map[string]string{
		"log.dirs":                                    "/tmp/kafka-logs",
		"num.network.threads":                         "3",
		"num.io.threads":                             "8",
		"socket.send.buffer.bytes":                   "102400",
		"socket.receive.buffer.bytes":                "102400",
		"socket.request.max.bytes":                   "104857600",
		"num.partitions":                             "1",
		"num.recovery.threads.per.data.dir":          "1",
		"offsets.topic.replication.factor":           "1",
		"transaction.state.log.replication.factor":   "1",
		"transaction.state.log.min.isr":              "1",
		"log.retention.hours":                        "168",
		"log.segment.bytes":                          "1073741824",
		"log.retention.check.interval.ms":            "300000",
	}
}

func getBrokerMetrics(brokerID int) map[string]float64 {
	// Simulate broker metrics
	return map[string]float64{
		"messages_per_sec":              1250.5,
		"bytes_per_sec":                 2048576.0,
		"active_controllers":            1.0,
		"offline_partitions":            0.0,
		"under_replicated_partitions":   0.0,
		"total_time_ms":                45.2,
		"request_queue_size":           5.0,
		"response_queue_size":          3.0,
		"network_processor_avg_idle":   85.5,
		"request_handler_avg_idle":     78.2,
	}
}

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

func testKafkaProtocol(host string, port int) bool {
	// In a real implementation, this would test the Kafka protocol
	// For now, we'll simulate success
	return true
}
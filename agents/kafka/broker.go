package main

import (
	"fmt"
	"net"
	"os"
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

	fmt.Println("📊 Kafka Broker Status:")

	brokers, err := kc.GetBrokerInfo()
	if err != nil {
		return err
	}

	for _, broker := range brokers {
		fmt.Printf("\n🖥️  Broker %d (%s:%d):\n", broker.ID, broker.Host, broker.Port)
		fmt.Printf("   Status: %s\n", getStatusEmoji(broker.Status)+" "+broker.Status)
		fmt.Printf("   Rack: %s\n", broker.Rack)
		fmt.Printf("   Leader: %t\n", broker.Leader)
		fmt.Printf("   Replicas: %d\n", broker.Replicas)
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

	fmt.Println("🏥 Kafka Broker Health Check:")

	brokers, err := kc.GetBrokerInfo()
	if err != nil {
		return err
	}

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

	fmt.Println("⚙️  Kafka Broker Configuration:")

	brokers, err := kc.GetBrokerInfo()
	if err != nil {
		return err
	}

	for _, broker := range brokers {
		fmt.Printf("\n🖥️  Broker %d (%s:%d):\n", broker.ID, broker.Host, broker.Port)

		config, err := kc.GetBrokerConfig(broker.ID)
		if err != nil {
			fmt.Printf("   ❌ Failed to get config: %v\n", err)
			continue
		}

		// Show key configuration parameters
		keyConfigs := []string{
			"broker.id", "listeners", "log.dirs", "num.network.threads", "num.io.threads",
			"socket.send.buffer.bytes", "socket.receive.buffer.bytes", "socket.request.max.bytes",
			"num.partitions", "num.recovery.threads.per.data.dir", "offsets.topic.replication.factor",
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
	// Note: Getting detailed metrics via Sarama is limited
	// This would typically require JMX or REST API access
	fmt.Println("📈 Kafka Broker Metrics:")
	fmt.Println("   Note: Detailed metrics require JMX or REST API access")
	fmt.Println("   Basic connectivity metrics available:")

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

	brokers, err := kc.GetBrokerInfo()
	if err != nil {
		return err
	}

	for _, broker := range brokers {
		fmt.Printf("\n🖥️  Broker %d (%s:%d):\n", broker.ID, broker.Host, broker.Port)
		fmt.Printf("   Status: %s\n", broker.Status)
		fmt.Printf("   Connection: %s\n", getConnectionStatus(broker.Status))
	}

	return nil
}

func testBrokerConnectivity() error {
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

	fmt.Println("🔌 Testing Kafka Broker Connectivity:")

	return kc.TestBrokerConnectivity()
}

// Helper functions
func getStatusEmoji(status string) string {
	switch status {
	case "Online":
		return "🟢"
	case "Offline":
		return "🔴"
	default:
		return "🟡"
	}
}

func getClusterHealth(online, total int) string {
	if total == 0 {
		return "🔴 Unknown"
	}
	percentage := float64(online) / float64(total) * 100
	if percentage >= 90 {
		return "🟢 Healthy"
	} else if percentage >= 70 {
		return "🟡 Warning"
	} else {
		return "🔴 Critical"
	}
}

func getConnectionStatus(status string) string {
	if status == "Online" {
		return "✅ Connected"
	}
	return "❌ Disconnected"
}

type HealthCheckResult struct {
	Success bool
	Error   string
}

func performHealthChecks(broker BrokerInfo) map[string]HealthCheckResult {
	results := make(map[string]HealthCheckResult)

	// TCP connectivity check
	host, port, err := parseBrokerAddress(broker.Host)
	if err != nil {
		results["TCP Connectivity"] = HealthCheckResult{
			Success: false,
			Error:   fmt.Sprintf("Invalid broker address: %v", err),
		}
		return results
	}

	// Test TCP connection
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		results["TCP Connectivity"] = HealthCheckResult{
			Success: false,
			Error:   fmt.Sprintf("Connection failed: %v", err),
		}
	} else {
		conn.Close()
		results["TCP Connectivity"] = HealthCheckResult{
			Success: true,
		}
	}

	// Kafka protocol check
	if broker.Status == "Online" {
		results["Kafka Protocol"] = HealthCheckResult{
			Success: true,
		}
	} else {
		results["Kafka Protocol"] = HealthCheckResult{
			Success: false,
			Error:   "Broker not responding to Kafka protocol",
		}
	}

	return results
}
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"regexp"
	"flag"
)


// Version information (set by GoReleaser or git)
var (
	version = "v0.1.0"
	commit  = getCommit()
	date    = getBuildDate()
)


func main() {
	// Handle flags
	var showVersion bool
	var displayHelp bool
	var message string
	var aiMode bool
	var troubleshoot bool
	var showStats bool
	var installAll bool
	var interactiveMode bool
	var adminMode string
	var kafkaBrokers string
	var kafkaCommandConfig string
	
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.BoolVar(&displayHelp, "help", false, "show help information")
	flag.StringVar(&message, "m", "", "natural language command message")
	flag.BoolVar(&aiMode, "ai", false, "use AI mode for advanced command generation")
	flag.BoolVar(&troubleshoot, "troubleshoot", false, "troubleshooting mode with context analysis")
	flag.BoolVar(&showStats, "stats", false, "show usage statistics")
	flag.BoolVar(&installAll, "install", false, "install all supported tools")
	flag.BoolVar(&interactiveMode, "o", false, "operations interactive mode")
	flag.StringVar(&adminMode, "admin", "", "enter admin mode for a specific service (e.g., 'kafka')")
	flag.StringVar(&kafkaBrokers, "brokers", "", "comma-separated list of Kafka brokers for admin mode")
	flag.StringVar(&kafkaCommandConfig, "command-config", "", "path to Kafka command config file for SSL/SASL")
	flag.Parse()

	if installAll {
		installAllTools()
		return
	}

	if adminMode != "" {
		switch adminMode {
		case "kafka":
			if kafkaBrokers == "" {
				fmt.Println("❌ ops0: --brokers flag is required for Kafka admin mode")
				os.Exit(1)
			}
			runKafkaAdminSession(kafkaBrokers, kafkaCommandConfig)
		default:
			fmt.Printf("❌ ops0: Unknown admin mode '%s'. Supported modes: kafka\n", adminMode)
			os.Exit(1)
		}
		return
	}

	if interactiveMode {
		runInteractiveSession()
		return
	}

	if displayHelp {
		showHelp()
		return
	}

	if showVersion {
		fmt.Printf("ops0 version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("built: %s\n", date)
		fmt.Printf("go version: %s\n", runtime.Version())
		fmt.Printf("platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return
	}

	if showStats {
		showCommandStats()
		return
	}

	// Check if message was provided
	if message == "" {
		fmt.Println("❌ ops0: No command message provided.")
		fmt.Println("💡 Use -m flag to specify a command, or -help for usage information.")
		showHelp()
		os.Exit(1)
	}

	// Initialize Claude if API key is available
	var claudeConfig *ClaudeConfig
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		model := os.Getenv("OPS0_AI_MODEL")
		if model == "" {
			model = "claude-3-5-sonnet-20241022"
		}
		claudeConfig = &ClaudeConfig{
			APIKey:    apiKey,
			Model:     model,
			MaxTokens: 1024,
		}
		fmt.Println("🧠 ops0: AI mode enabled")
	} else if aiMode {
		fmt.Println("❌ ops0: AI mode requested but ANTHROPIC_API_KEY not found")
		fmt.Println("💡 Set your API key: export ANTHROPIC_API_KEY=your_key_here")
		os.Exit(1)
	}

	fmt.Printf("🤖 ops0: Analyzing your request: \"%s\"\n\n", message)

	var suggestion *CommandSuggestion

	// Try AI-powered analysis first if available
	if claudeConfig != nil {
		if troubleshoot {
			suggestion = handleTroubleshooting(claudeConfig, message)
		} else {
			suggestion = getAISuggestion(claudeConfig, message)
		}
	}

	// Post-process for log analysis intent if needed
	if suggestion != nil && suggestion.Tool == "kubectl" && strings.Contains(suggestion.Command, "logs") {
		msgLower := strings.ToLower(message)
		if strings.Contains(msgLower, "analyze") || strings.Contains(msgLower, "review") {
			suggestion.Intent = "analyze_logs"
		}
	}

	// Post-process for log file analysis intent if needed
	if suggestion != nil && strings.Contains(suggestion.Command, ".log") {
		msgLower := strings.ToLower(message)
		if strings.Contains(msgLower, "analyze") || strings.Contains(msgLower, "review") ||
		   strings.Contains(msgLower, "check") || strings.Contains(msgLower, "summarize") || strings.Contains(msgLower, "inspect") {
			suggestion.Intent = "analyze_logs"
			// Use a safe preview command for analysis, not tail -f
			re := regexp.MustCompile(`([^-\s]+\.log)`)
			if m := re.FindStringSubmatch(suggestion.Command); len(m) > 1 {
				suggestion.Command = "tail -n 100 " + m[1]
			}
		}
	}

	// Fallback to rule-based parsing if AI didn't work or isn't available
	if suggestion == nil {
		suggestion = ParseIntent(message)
	}
	
	if suggestion == nil {
		fmt.Println("❌ ops0: I couldn't understand your request. Try being more specific about what you want to do.")
		if claudeConfig == nil {
			fmt.Println("💡 For better understanding, set ANTHROPIC_API_KEY to enable AI mode")
		}
		return
	}

	// Present the suggestion interactively
	handleInteraction(suggestion)
}

func showHelp() {
	fmt.Println("ops0 - AI-Powered Natural Language DevOps CLI")
	fmt.Printf("Version: %s\n\n", version)

	// Basic Usage
	fmt.Println("📋 Usage:")
	fmt.Println("  ops0 -m \"your natural language command\"")
	fmt.Println("  ops0 -o")
	fmt.Println("  ops0 -m \"command\" -ai")
	fmt.Println("  ops0 -m \"error description\" -troubleshoot")
	fmt.Println("  ops0 -version")
	fmt.Println("  ops0 -help")

	// Flags
	fmt.Println("\n🚩 Flags:")
	fmt.Println("  -m           Natural language command message (required)")
	fmt.Println("  -o           Enter interactive operations mode")
	fmt.Println("  -ai          Enable AI mode for advanced command generation")
	fmt.Println("  -troubleshoot Enable troubleshooting mode with context analysis")
	fmt.Println("  -version     Show version information")
	fmt.Println("  -help        Show this help message")
	fmt.Println("  -install     Install all supported tools and display their versions")

	// Admin Modes
	fmt.Println("\n🔒 Admin Modes:")
	fmt.Println("  Enter an interactive admin session for a specific service.")
	fmt.Println("\n  Kafka Admin Mode:")
	fmt.Println("    Usage: ops0 --admin kafka --brokers <broker_list>")
	fmt.Println("    Flags:")
	fmt.Println("      --admin kafka              Enter Kafka admin mode.")
	fmt.Println("      --brokers <list>           Comma-separated list of Kafka brokers (required).")
	fmt.Println("      --command-config <path>    Path to client config file for SSL/SASL.")
	fmt.Println("    Example:")
	fmt.Println("      ops0 --admin kafka --brokers localhost:9092")
	fmt.Println("      ops0 --admin kafka --brokers ssl-broker:9093 --command-config client.properties")

	// Supported Tools
	fmt.Println("\n🛠️  Supported Tools:")
	
	// Terraform
	fmt.Println("\n  Terraform (Infrastructure as Code):")
	fmt.Println("    • terraform plan     - Show infrastructure changes")
	fmt.Println("    • terraform apply    - Apply infrastructure changes")
	fmt.Println("    • terraform destroy  - Destroy infrastructure")
	fmt.Println("    Examples:")
	fmt.Println("      ops0 -m \"plan my infrastructure changes\"")
	fmt.Println("      ops0 -m \"apply terraform configuration\"")

	// Ansible
	fmt.Println("\n  Ansible (Configuration Management):")
	fmt.Println("    • ansible-playbook   - Run Ansible playbooks")
	fmt.Println("    • ansible-playbook --check - Dry run playbooks")
	fmt.Println("    Examples:")
	fmt.Println("      ops0 -m \"run my ansible playbook\"")
	fmt.Println("      ops0 -m \"check ansible changes\"")

	// Kubernetes
	fmt.Println("\n  Kubernetes (Container Orchestration):")
	fmt.Println("    • kubectl get pods   - List pods")
	fmt.Println("    • kubectl apply      - Apply manifests")
	fmt.Println("    • kubectl delete     - Delete resources")
	fmt.Println("    • kubectl logs       - View pod logs")
	fmt.Println("    Examples:")
	fmt.Println("      ops0 -m \"show me my pods\"")
	fmt.Println("      ops0 -m \"deploy to kubernetes\"")

	// Docker
	fmt.Println("\n  Docker (Containerization):")
	fmt.Println("    • docker ps          - List containers")
	fmt.Println("    • docker build       - Build images")
	fmt.Println("    • docker images      - List images")
	fmt.Println("    Examples:")
	fmt.Println("      ops0 -m \"show running containers\"")
	fmt.Println("      ops0 -m \"build docker image\"")

	// AWS CLI
	fmt.Println("\n  AWS CLI (Amazon Web Services):")
	fmt.Println("    • aws ec2            - EC2 operations")
	fmt.Println("    • aws s3             - S3 operations")
	fmt.Println("    Examples:")
	fmt.Println("      ops0 -m \"list ec2 instances\"")
	fmt.Println("      ops0 -m \"show s3 buckets\"")

	// AI Mode
	fmt.Println("\n🧠 AI Mode:")
	fmt.Println("  Enable AI mode for advanced features:")
	fmt.Println("  1. Get API key from console.anthropic.com")
	fmt.Println("  2. Export key: export ANTHROPIC_API_KEY=your_key_here")
	fmt.Println("  3. Use -ai flag: ops0 -m \"your command\" -ai")
	fmt.Println("\n  AI mode benefits:")
	fmt.Println("    • Better natural language understanding")
	fmt.Println("    • Context-aware suggestions")
	fmt.Println("    • Advanced troubleshooting")
	fmt.Println("    • Support for complex scenarios")

	// Dry Run Support
	fmt.Println("\n🔍 Dry Run Support:")
	fmt.Println("  Available for these operations:")
	fmt.Println("    • Terraform: plan before apply/destroy")
	fmt.Println("    • Ansible: --check flag")
	fmt.Println("    • Kubernetes: --dry-run=client flag")
	fmt.Println("  Will be offered automatically when available")

	// Examples
	fmt.Println("\n💡 More Examples:")
	fmt.Println("  Infrastructure:")
	fmt.Println("    ops0 -m \"plan my terraform changes\"")
	fmt.Println("    ops0 -m \"apply infrastructure with approval\"")
	fmt.Println("\n  Kubernetes:")
	fmt.Println("    ops0 -m \"show pods in namespace monitoring\"")
	fmt.Println("    ops0 -m \"deploy app to production namespace\"")
	fmt.Println("\n  Troubleshooting:")
	fmt.Println("    ops0 -m \"why are my pods crashing\" -troubleshoot")
	fmt.Println("    ops0 -m \"check why terraform is failing\" -troubleshoot")

	fmt.Println("\n📚 Documentation:")
	fmt.Println("  Full documentation: https://github.com/ops0-ai/ops0-cli")
	fmt.Println("  Report issues: https://github.com/ops0-ai/ops0-cli/issues")
}


func formatSection(title string, content []string) string {
	var output strings.Builder
	
	// Title with underline
	output.WriteString("\n" + blue + bold + title + reset + "\n")
	output.WriteString(blue + strings.Repeat("─", len(title)) + reset + "\n")
	
	// Content with bold keys
	for _, line := range content {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			output.WriteString(bold + parts[0] + ":" + reset + parts[1] + "\n")
		} else {
			output.WriteString(line + "\n")
		}
	}
	
	return output.String()
}


func getCommit() string {
	if c := os.Getenv("COMMIT"); c != "" {
		return c
	}
	
	if cmd := exec.Command("git", "rev-parse", "--short", "HEAD"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			return strings.TrimSpace(string(output))
		}
	}
	
	return "none"
}

func getBuildDate() string {
	if d := os.Getenv("BUILD_DATE"); d != "" {
		return d
	}
	
	return "unknown"
}


func getUserConfirmation() bool {
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	
	return response == "y" || response == "yes"
}



func mapKeys(m map[string]struct{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func extractIP(s string) string {
	re := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	match := re.FindString(s)
	return match
}

func extractProjectName(msg string) string {
	re := regexp.MustCompile(`(?i)project\s+([a-zA-Z0-9_-]+)`)
	match := re.FindStringSubmatch(msg)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}


func extractPackageCommand(input string) string {
	input = strings.ToLower(input)
	
	// Detect package manager
	var pkgManager string
	if isCommandAvailable("apt") {
		pkgManager = "apt"
	} else if isCommandAvailable("yum") {
		pkgManager = "yum"
	} else if isCommandAvailable("dnf") {
		pkgManager = "dnf"
	} else {
		pkgManager = "apt" // Default to apt
	}
	
	// Extract package name if present
	re := regexp.MustCompile(`(install|update|upgrade)\s+([a-zA-Z0-9-]+)`)
	match := re.FindStringSubmatch(input)
	
	if strings.Contains(input, "update") || strings.Contains(input, "upgrade") {
		return fmt.Sprintf("sudo %s update && sudo %s upgrade -y", pkgManager, pkgManager)
	}
	
	if len(match) > 2 {
		return fmt.Sprintf("sudo %s install -y %s", pkgManager, match[2])
	}
	
	return fmt.Sprintf("sudo %s update", pkgManager)
}

func extractServiceCommand(input string) string {
	input = strings.ToLower(input)
	
	// Extract service name and action
	re := regexp.MustCompile(`(start|stop|restart|status)\s+([a-zA-Z0-9-]+)`)
	match := re.FindStringSubmatch(input)
	
	if len(match) > 2 {
		action := match[1]
		service := match[2]
		return fmt.Sprintf("sudo systemctl %s %s", action, service)
	}
	
	return "systemctl list-units --type=service --state=running"
}

func extractSystemMonitorCommand(input string) string {
	input = strings.ToLower(input)
	
	if strings.Contains(input, "memory") || strings.Contains(input, "ram") {
		return "free -h"
	}
	
	if strings.Contains(input, "disk") || strings.Contains(input, "storage") || strings.Contains(input, "df") {
		return "df -h"
	}
	
	if strings.Contains(input, "cpu") || strings.Contains(input, "processor") || strings.Contains(input, "top") {
		return "top -b -n 1"
	}
	
	// Default to showing all system resources
	return "echo '=== Memory Usage ===' && free -h && echo -e '\n=== Disk Usage ===' && df -h && echo -e '\n=== CPU Usage ===' && top -b -n 1"
}

func extractLogCommand(input string) string {
	input = strings.ToLower(input)
	
	if strings.Contains(input, "journal") || strings.Contains(input, "system") {
		return "sudo journalctl -n 50"
	}
	
	if strings.Contains(input, "tail") {
		return "sudo tail -f /var/log/syslog"
	}
	
	return "sudo journalctl -n 50"
}

func isCommandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}


func simpleLogAnalysis(logs string) string {
	lines := strings.Split(logs, "\n")
	errors, warns := []string{}, []string{}
	for _, l := range lines {
		if strings.Contains(strings.ToLower(l), "error") {
			errors = append(errors, l)
		}
		if strings.Contains(strings.ToLower(l), "warn") {
			warns = append(warns, l)
		}
	}
	if len(errors) == 0 && len(warns) == 0 {
		return "Logs look fine, no errors or warnings detected."
	}
	var b strings.Builder
	if len(errors) > 0 {
		b.WriteString("Errors found:\n")
		for _, e := range errors {
			b.WriteString("  " + e + "\n")
		}
	}
	if len(warns) > 0 {
		b.WriteString("Warnings found:\n")
		for _, w := range warns {
			b.WriteString("  " + w + "\n")
		}
	}
	b.WriteString("\nRecommendation: Investigate the above issues.\n")
	return b.String()
}

// findCommand checks for a command in PATH, then in common locations.
// It returns the full path to the command if found, and an error indicating status.
// Error can be 'not_found' or 'found_not_in_path'. The path returned on 'found_not_in_path'
// is the location where the command was found.
func findCommand(cmd string) (string, error) {
	// 1. Check PATH first. If found, we are good.
	path, err := exec.LookPath(cmd)
	if err == nil {
		return path, nil
	}

	// 2. If not in PATH, check common alternative locations on macOS.
	if runtime.GOOS == "darwin" {
		commonPaths := []string{
			"/opt/homebrew/bin", // Apple Silicon
			"/usr/local/bin",    // Intel Macs
		}
		for _, p := range commonPaths {
			fullPath := filepath.Join(p, cmd)
			if _, err := os.Stat(fullPath); err == nil {
				// Found it, but it wasn't in the system PATH.
				return fullPath, fmt.Errorf("found_not_in_path")
			}
		}
	}

	// 3. Really not found.
	return "", fmt.Errorf("not_found")
}
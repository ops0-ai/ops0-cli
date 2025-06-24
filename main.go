package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"regexp"
	"flag"
	"time"
)



// ANSI escape codes for text formatting
const (
	bold      = "\033[1m"
	reset     = "\033[0m"
	blue      = "\033[34m"
	green     = "\033[32m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	underline = "\033[4m"
)

// Version information (set by GoReleaser or git)
var (
	version = "v0.1.0"
	commit  = getCommit()
	date    = getBuildDate()
)

// Claude API configuration
type ClaudeConfig struct {
	APIKey string
	Model  string
	MaxTokens int
}

// Claude API request/response structures
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []ClaudeMessage `json:"messages"`
	System    string          `json:"system"`
}

type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type Tool struct {
	Name        string
	CheckCmd    string
	InstallCmd  string
	IsInstalled bool
}

type CommandSuggestion struct {
	Tool        string
	Command     string
	DryRunCommand string  // Command to use for dry run
	Description string
	Intent      string
	Confidence  float64
	AIGenerated bool
	HasDryRun   bool     // Whether this command supports dry run
}

type cmdCount struct {
	cmd   string
	count int
}

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



func getAISuggestion(config *ClaudeConfig, userInput string) *CommandSuggestion {
	systemPrompt := `You are ops0, an AI-powered DevOps CLI assistant. Your job is to translate natural language requests into specific DevOps commands.

You support these tools: terraform, ansible, kubectl, docker, helm, aws-cli, gcloud, azure-cli, system_admin.

For system monitoring and resource usage requests:
- If the request mentions "device", "machine", "local", "my", or "system", use system_admin tool
- Use system_admin for memory, CPU, disk usage, and system logs
- Only use docker/k8s tools if explicitly mentioning containers or clusters

Respond with a JSON object in this exact format:
{
  "tool": "system_admin",
  "command": "free -h",
  "dry_run_command": "",
  "description": "This will show memory usage on your local machine",
  "intent": "monitor local system resources",
  "confidence": 0.95,
  "has_dry_run": false
}

Rules:
- Only suggest commands for tools that are commonly available
- Prefer safe, read-only commands when possible
- Include helpful descriptions
- Set confidence between 0-1 based on how certain you are
- For commands that modify state, provide a dry run command if available
- If you can't understand the request, return null`

	response := callClaude(config, systemPrompt, userInput)
	if response == "" {
		return nil
	}

	var suggestion CommandSuggestion
	if err := json.Unmarshal([]byte(response), &suggestion); err != nil {
		fmt.Printf("⚠️  ops0: AI response parsing error, falling back to rule-based parsing\n")
		return nil
	}

	suggestion.AIGenerated = true
	return &suggestion
}

func handleTroubleshooting(config *ClaudeConfig, problem string) *CommandSuggestion {
	context := gatherSystemContext()
	
	systemPrompt := `You are ops0, an AI-powered DevOps troubleshooting assistant. The user is experiencing a problem and needs help.

Analyze the problem and system context, then suggest the best diagnostic or fix command.

Respond with a JSON object:
{
  "tool": "kubectl",
  "command": "kubectl describe pods",
  "description": "This will show detailed information about pod issues",
  "intent": "diagnose pod problems",
  "confidence": 0.9
}

Focus on diagnostic commands first and safe operations.`

	prompt := fmt.Sprintf("Problem: %s\n\nSystem Context:\n%s", problem, context)
	response := callClaude(config, systemPrompt, prompt)
	
	if response == "" {
		return nil
	}

	var suggestion CommandSuggestion
	if err := json.Unmarshal([]byte(response), &suggestion); err != nil {
		return nil
	}

	suggestion.AIGenerated = true
	return &suggestion
}

func gatherSystemContext() string {
	var context strings.Builder
	
	tools := []string{"terraform", "kubectl", "docker", "ansible", "helm", "aws", "gcloud", "az"}
	context.WriteString("Available tools:\n")
	
	for _, tool := range tools {
		if isToolInstalled(tool) {
			if version := getToolVersion(tool); version != "" {
				context.WriteString(fmt.Sprintf("- %s: %s\n", tool, version))
			} else {
				context.WriteString(fmt.Sprintf("- %s: installed\n", tool))
			}
		}
	}
	
	context.WriteString("\nProject context:\n")
	files := []string{"terraform.tf", "main.tf", "Dockerfile", "docker-compose.yml", 
					 "kubernetes.yaml", "k8s.yaml", "playbook.yml", "ansible.yml"}
	
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			context.WriteString(fmt.Sprintf("- Found: %s\n", file))
		}
	}
	
	if pwd, err := os.Getwd(); err == nil {
		context.WriteString(fmt.Sprintf("- Working directory: %s\n", pwd))
	}
	
	return context.String()
}

func callClaude(config *ClaudeConfig, systemPrompt, userMessage string) string {
	request := ClaudeRequest{
		Model:     config.Model,
		MaxTokens: config.MaxTokens,
		System:    systemPrompt,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: userMessage,
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("⚠️  ops0: Error preparing AI request: %v\n", err)
		return ""
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("⚠️  ops0: Error creating AI request: %v\n", err)
		return ""
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("⚠️  ops0: Error calling AI service: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("⚠️  ops0: Error reading AI response: %v\n", err)
		return ""
	}

	if resp.StatusCode != 200 {
		fmt.Printf("⚠️  ops0: AI service error (status %d): %s\n", resp.StatusCode, string(body))
		return ""
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		fmt.Printf("⚠️  ops0: Error parsing AI response: %v\n", err)
		return ""
	}

	if len(claudeResp.Content) > 0 {
		return claudeResp.Content[0].Text
	}

	return ""
}

func isToolInstalled(tool string) bool {
	cmd := exec.Command("which", tool)
	return cmd.Run() == nil
}

func getToolVersion(tool string) string {
	var cmd *exec.Cmd
	switch tool {
	case "terraform":
		cmd = exec.Command("terraform", "version")
	case "kubectl":
		cmd = exec.Command("kubectl", "version", "--client", "--short")
	case "docker":
		cmd = exec.Command("docker", "--version")
	case "ansible":
		cmd = exec.Command("ansible", "--version")
	default:
		cmd = exec.Command(tool, "--version")
	}
	
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	
	return ""
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



func getToolDisplayName(toolName string) string {
	switch toolName {
	case "aws", "aws-cli":
		return "AWS CLI"
	case "gcloud":
		return "Google Cloud SDK"
	case "az":
		return "Azure CLI"
	case "kubectl":
		return "Kubernetes CLI"
	case "kafka":
		return "Apache Kafka"
	default:
		return strings.Title(toolName)
	}
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

func checkToolInstalled(tool *Tool) bool {
	cmd := exec.Command("sh", "-c", tool.CheckCmd+" > /dev/null 2>&1")
	return cmd.Run() == nil
}

func getInstallCommand(toolName string) string {
	switch toolName {
	case "terraform":
		if runtime.GOOS == "darwin" {
			if runtime.GOARCH == "arm64" {
				return "arch -arm64 brew install terraform"
			}
			return "brew install terraform"
		}
		return "curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add - && sudo apt-add-repository \"deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main\" && sudo apt-get update && sudo apt-get install terraform"
	case "ansible":
		if runtime.GOOS == "darwin" {
			if runtime.GOARCH == "arm64" {
				return "arch -arm64 brew install ansible"
			}
			return "brew install ansible"
		}
		return "sudo apt-get update && sudo apt-get install ansible"
	case "kubectl":
		if runtime.GOOS == "darwin" {
			if runtime.GOARCH == "arm64" {
				return "arch -arm64 brew install kubectl"
			}
			return "brew install kubectl"
		}
		return "curl -LO \"https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl\" && sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl"
	case "docker":
		if runtime.GOOS == "darwin" {
			return "echo 'Please install Docker Desktop from https://www.docker.com/products/docker-desktop/' && open 'https://www.docker.com/products/docker-desktop/'"
		}
		return "curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh"
	case "helm":
		if runtime.GOOS == "darwin" {
			if runtime.GOARCH == "arm64" {
				return "arch -arm64 brew install helm"
			}
			return "brew install helm"
		}
		return "curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash"
	case "aws":
		if runtime.GOOS == "darwin" {
			// Use direct installer for macOS to avoid Homebrew architecture issues
			if runtime.GOARCH == "arm64" {
				return "curl \"https://awscli.amazonaws.com/AWSCLIV2-arm64.pkg\" -o \"AWSCLIV2.pkg\" && sudo installer -pkg AWSCLIV2.pkg -target /"
			}
			return "curl \"https://awscli.amazonaws.com/AWSCLIV2.pkg\" -o \"AWSCLIV2.pkg\" && sudo installer -pkg AWSCLIV2.pkg -target /"
		}
		return "curl \"https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip\" -o \"awscliv2.zip\" && unzip awscliv2.zip && sudo ./aws/install"
	case "gcloud":
		if runtime.GOOS == "darwin" {
			if runtime.GOARCH == "arm64" {
				return "arch -arm64 brew install google-cloud-sdk"
			}
			return "brew install google-cloud-sdk"
		}
		return "curl https://sdk.cloud.google.com | bash && exec -l $SHELL"
	case "az":
		if runtime.GOOS == "darwin" {
			if runtime.GOARCH == "arm64" {
				return "arch -arm64 brew install azure-cli"
			}
			return "brew install azure-cli"
		}
		return "curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash"
	case "kafka":
		if runtime.GOOS == "darwin" {
			if runtime.GOARCH == "arm64" {
				return "arch -arm64 brew install kafka"
			}
			return "brew install kafka"
		}
		// For Linux, download from Apache, extract, and symlink binaries
		return "echo 'Downloading and installing Apache Kafka...' && KAFKA_VERSION=\"3.7.0\" && SCALA_VERSION=\"2.13\" && curl -L \"https://downloads.apache.org/kafka/${KAFKA_VERSION}/kafka_${SCALA_VERSION}-${KAFKA_VERSION}.tgz\" -o kafka.tgz && tar -xzf kafka.tgz && sudo mv kafka_${SCALA_VERSION}-${KAFKA_VERSION} /usr/local/kafka && sudo ln -s /usr/local/kafka/bin/* /usr/local/bin/ && rm kafka.tgz && echo 'Kafka installed to /usr/local/kafka. Binaries symlinked to /usr/local/bin.'"
	default:
		return ""
	}
}

func installTool(tool *Tool) bool {
	if tool.InstallCmd == "" {
		fmt.Printf("❌ ops0: Don't know how to install %s on this system.\n", tool.Name)
		fmt.Printf("🔍 Debug: Tool name = '%s', OS = %s\n", tool.Name, runtime.GOOS)
		return false
	}
	
	fmt.Printf("🔧 ops0: Installing %s...\n", tool.Name)
	cmd := exec.Command("sh", "-c", tool.InstallCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run() == nil
}

func findPlaybookFile() string {
	candidates := []string{"playbook.yml", "site.yml", "main.yml", "deploy.yml"}
	
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	
	return ""
}

func getUserConfirmation() bool {
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	
	return response == "y" || response == "yes"
}

// Log every executed command to ~/.ops0-cli-stats.log
func logCommandStat(tool, command string) {
	usr, err := user.Current()
	username := "unknown"
	if err == nil {
		username = usr.Username
	}
	fmt.Fprintf(os.Stderr, "LOGGING: %s %s %s\n", username, tool, command)
	home := os.Getenv("HOME")
	if home == "" && err == nil {
		home = usr.HomeDir
	}
	if home == "" {
		fmt.Fprintln(os.Stderr, "Could not determine home directory for stats logging.")
		return
	}
	logPath := home + "/.ops0-cli-stats.log"
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open stats log file: %v\n", err)
		return
	}
	defer f.Close()
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("%s|%s|%s|%s\n", timestamp, username, tool, command)
	f.WriteString(line)
}

// Show stats from ~/.ops0-cli-stats.log
func showCommandStats() {
	usr, err := user.Current()
	home := os.Getenv("HOME")
	if home == "" && err == nil {
		home = usr.HomeDir
	}
	if home == "" {
		fmt.Println("Could not determine user home directory.")
		return
	}
	logPath := home + "/.ops0-cli-stats.log"
	f, err := os.Open(logPath)
	if err != nil {
		fmt.Println("No command stats found yet. Run some commands first!")
		return
	}
	defer f.Close()

	total := 0
	toolCounts := make(map[string]int)
	var lastUsed string
	var mostUsedTool string
	maxCount := 0
	commandCounts := make(map[string]int)
	operationCounts := make(map[string]map[string]int) // tool -> op -> count
	userSet := make(map[string]struct{})

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "|", 4)
		if len(parts) != 4 {
			continue
		}
		total++
		ts, user, tool, command := parts[0], parts[1], parts[2], parts[3]
		userSet[user] = struct{}{}
		toolCounts[tool]++
		lastUsed = ts
		commandCounts[command]++
		if toolCounts[tool] > maxCount {
			maxCount = toolCounts[tool]
			mostUsedTool = tool
		}
		// Operation classification
		if _, ok := operationCounts[tool]; !ok {
			operationCounts[tool] = make(map[string]int)
		}
		var op string
		switch tool {
		case "ansible":
			if strings.Contains(command, "playbook") {
				op = "run playbook"
			} else {
				op = "ad-hoc command"
			}
		case "kubectl":
			if strings.Contains(command, "get pods") {
				op = "get pods"
			} else if strings.Contains(command, "apply") {
				op = "apply"
			} else if strings.Contains(command, "delete") {
				op = "delete"
			} else {
				op = "other"
			}
		case "terraform":
			if strings.Contains(command, "plan") {
				op = "plan"
			} else if strings.Contains(command, "apply") {
				op = "apply"
			} else if strings.Contains(command, "destroy") {
				op = "destroy"
			} else {
				op = "other"
			}
		case "docker":
			if strings.Contains(command, "ps") {
				op = "ps"
			} else if strings.Contains(command, "build") {
				op = "build"
			} else if strings.Contains(command, "images") {
				op = "images"
			} else {
				op = "other"
			}
		case "aws":
			if strings.Contains(command, "ec2") {
				op = "ec2"
			} else if strings.Contains(command, "s3") {
				op = "s3"
			} else {
				op = "other"
			}
		default:
			op = "other"
		}
		operationCounts[tool][op]++
	}
	if total == 0 {
		fmt.Println("No command stats found yet. Run some commands first!")
		return
	}
	fmt.Println("\n📊 ops0 Command Usage Stats")
	fmt.Println("══════════════════════════")
	fmt.Printf("User(s): %s\n", strings.Join(mapKeys(userSet), ", "))
	fmt.Printf("Total Commands Run: %d\n", total)
	fmt.Println("Per-Tool Usage:")
	for tool, count := range toolCounts {
		fmt.Printf("  %s: %d\n", tool, count)
	}
	fmt.Printf("Most Used Tool: %s (%d times)\n", mostUsedTool, maxCount)
	fmt.Printf("Last Used: %s\n", lastUsed)
	fmt.Println("\nOperation Types per Tool:")
	for tool, ops := range operationCounts {
		fmt.Printf("  %s:\n", tool)
		for op, count := range ops {
			fmt.Printf("    %s: %d\n", op, count)
		}
	}
	fmt.Println("\nTop 10 Commands:")
	topCmds := topNCommands(commandCounts, 10)
	for i, pair := range topCmds {
		fmt.Printf("  %d. %s (%d times)\n", i+1, pair.cmd, pair.count)
	}
}


func topNCommands(m map[string]int, n int) []cmdCount {
	var arr []cmdCount
	for k, v := range m {
		arr = append(arr, cmdCount{k, v})
	}
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].count > arr[j].count
	})
	if len(arr) > n {
		return arr[:n]
	}
	return arr
}

func mapKeys(m map[string]struct{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func generateAnsibleProjectAIWithFilenames(userMsg string) (string, string, string, string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", "", "", "", fmt.Errorf("No ANTHROPIC_API_KEY set")
	}
	prompt := `You are an expert DevOps assistant. Given the following user request, generate:
1. A complete Ansible playbook YAML (for playbook file)
2. A valid Ansible inventory file (for inventory file)

Respond in this format:
---PLAYBOOK FILE---
<playbook filename>
---PLAYBOOK---
<playbook yaml>
---INVENTORY FILE---
<inventory filename>
---INVENTORY---
<inventory content>

User request: ` + userMsg
	claudeConfig := &ClaudeConfig{
		APIKey: apiKey,
		Model:  "claude-3-5-sonnet-20241022",
		MaxTokens: 1024,
	}
	response := callClaude(claudeConfig, prompt, "")
	if response == "" {
		return "", "", "", "", fmt.Errorf("AI did not return a response")
	}
	playbookContent, inventoryContent, playbookFile, inventoryFile := parseAnsibleAIResponseWithFilenames(response)
	if playbookFile == "" {
		playbookFile = "playbook.yml"
	}
	if inventoryFile == "" {
		inventoryFile = "inventory.yml"
	}
	return playbookContent, inventoryContent, playbookFile, inventoryFile, nil
}

func parseAnsibleAIResponseWithFilenames(resp string) (string, string, string, string) {
	playbook := ""
	inventory := ""
	playbookFile := ""
	inventoryFile := ""
	pfStart := strings.Index(resp, "---PLAYBOOK FILE---")
	pStart := strings.Index(resp, "---PLAYBOOK---")
	ifStart := strings.Index(resp, "---INVENTORY FILE---")
	iStart := strings.Index(resp, "---INVENTORY---")
	if pfStart != -1 && pStart != -1 {
		playbookFile = strings.TrimSpace(resp[pfStart+len("---PLAYBOOK FILE---"):pStart])
	}
	if pStart != -1 && ifStart != -1 {
		playbook = strings.TrimSpace(resp[pStart+len("---PLAYBOOK---"):ifStart])
	}
	if ifStart != -1 && iStart != -1 {
		inventoryFile = strings.TrimSpace(resp[ifStart+len("---INVENTORY FILE---"):iStart])
	}
	if iStart != -1 {
		inventory = strings.TrimSpace(resp[iStart+len("---INVENTORY---"):])
	}
	return playbook, inventory, playbookFile, inventoryFile
}

func generateAnsibleProjectTemplate(userMsg string) (string, string, error) {
	// Simple fallback: extract project name, group, host (very basic)
	project := "ansible-project"
	group := "web"
	host := "127.0.0.1"
	if strings.Contains(userMsg, "nginx") {
		group = "nginx"
	}
	if ip := extractIP(userMsg); ip != "" {
		host = ip
	}
	playbook := fmt.Sprintf(`- name: %s
  hosts: %s
  become: yes
  tasks:
    - name: Install nginx
      apt:
        name: nginx
        state: present
      when: ansible_os_family == 'Debian'
    - name: Restart nginx
      service:
        name: nginx
        state: restarted
    - name: Create symlink
      file:
        src: /some/source
        dest: /some/dest
        state: link
`, project, group)
	inventory := fmt.Sprintf(`[%s]
%s
`, group, host)
	return playbook, inventory, nil
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

// Helper to parse AI description for file names and YAML blocks
func parseAnsibleFilesFromAIDescription(desc string) (map[string]string, error) {
	files := make(map[string]string)
	lines := strings.Split(desc, "\n")
	var currentFile string
	var currentContent []string
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasSuffix(line, "with:") && !strings.Contains(line, "Then ") {
			if currentFile != "" && len(currentContent) > 0 {
				files[currentFile] = strings.Join(currentContent, "\n")
			}
			currentFile = strings.TrimSuffix(line, " with:")
			currentContent = []string{}
			continue
		}
		if currentFile != "" {
			if line == "AI Confidence: 85%" || strings.HasPrefix(line, "Would you like to execute") || strings.HasPrefix(line, "Command:") {
				files[currentFile] = strings.Join(currentContent, "\n")
				currentFile = ""
				currentContent = []string{}
				continue
			}
			currentContent = append(currentContent, lines[i])
		}
	}
	if currentFile != "" && len(currentContent) > 0 {
		files[currentFile] = strings.Join(currentContent, "\n")
	}
	return files, nil
}

func findAnsiblePlaybookAndInventory(files map[string]string) (string, string) {
	playbookFile := ""
	inventoryFile := ""
	for fname := range files {
		if strings.Contains(fname, "playbook") || strings.HasSuffix(fname, ".yml") && playbookFile == "" {
			playbookFile = fname
		}
		if strings.Contains(fname, "inventory") || strings.HasPrefix(fname, "inv") {
			inventoryFile = fname
		}
	}
	if playbookFile == "" {
		playbookFile = "playbook.yml"
	}
	if inventoryFile == "" {
		inventoryFile = "inventory.yml"
	}
	return playbookFile, inventoryFile
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

func getClaudeConfigIfAvailable() *ClaudeConfig {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil
	}
	model := os.Getenv("OPS0_AI_MODEL")
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	return &ClaudeConfig{
		APIKey:    apiKey,
		Model:     model,
		MaxTokens: 1024,
	}
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

func installAllTools() {
	tools := []string{"terraform", "ansible", "kubectl", "docker", "helm", "aws", "gcloud", "az"}
	fmt.Println("🔧 Installing all supported tools...")
	for _, name := range tools {
		tool := &Tool{
			Name:       name,
			CheckCmd:   name + " --version",
			InstallCmd: getInstallCommand(name),
		}
		if checkToolInstalled(tool) {
			fmt.Printf("✅ %s is already installed.\n", getToolDisplayName(name))
			continue
		}
		fmt.Printf("🔧 Installing %s...\n", getToolDisplayName(name))
		if installTool(tool) {
			fmt.Printf("✅ %s installed successfully!\n", getToolDisplayName(name))
		} else {
			fmt.Printf("❌ Failed to install %s. Please install it manually.\n", getToolDisplayName(name))
		}
	}
	fmt.Println("🎉 All tools processed.")

	// Display table of installed tools and versions
	fmt.Println("\n📦 Installed Tools:")
	fmt.Println("────────────────────────────────────────────")
	fmt.Printf("%-18s | %-20s\n", "Tool", "Version")
	fmt.Println(strings.Repeat("-", 42))
	for _, name := range tools {
		ver := getToolVersion(name)
		if ver == "" {
			ver = "Not installed"
		}
		fmt.Printf("%-18s | %-20s\n", getToolDisplayName(name), ver)
	}
	fmt.Println(strings.Repeat("-", 42))
}

func runInteractiveSession() {
	fmt.Println("🔄 ops0 Interactive Operations Mode (type 'quit' or 'exit' to leave)")
	reader := bufio.NewReader(os.Stdin)
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
		fmt.Println("🧠 AI mode enabled in interactive session")
	}
	for {
		fmt.Print("ops0> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "quit" || input == "exit" {
			fmt.Println("👋 Exiting ops0 operations mode.")
			break
		}
		if input == "" {
			continue
		}
		var suggestion *CommandSuggestion
		if claudeConfig != nil {
			suggestion = getAISuggestion(claudeConfig, input)
		}
		if suggestion == nil {
			suggestion = ParseIntent(input)
		}
		if suggestion != nil {
			// Post-process for log file analysis intent if needed
			if strings.Contains(suggestion.Command, ".log") {
				msgLower := strings.ToLower(input)
				if strings.Contains(msgLower, "analyze") || strings.Contains(msgLower, "debug") ||
				   strings.Contains(msgLower, "review") || strings.Contains(msgLower, "check") ||
				   strings.Contains(msgLower, "summarize") || strings.Contains(msgLower, "inspect") {
					suggestion.Intent = "analyze_logs"
					// Use a safe preview command for analysis, not tail -f
					re := regexp.MustCompile(`([^-\s]+\.log)`)
					if m := re.FindStringSubmatch(suggestion.Command); len(m) > 1 {
						suggestion.Command = "tail -n 100 " + m[1]
					}
				}
			}
			if suggestion.Intent == "analyze_logs" {
				// Log analysis flow: preview, prompt for AI, show summary
				fmt.Println("\n--- Log Preview ---")
				cmd := exec.Command("bash", "-c", suggestion.Command)
				output, err := cmd.CombinedOutput()
				if err != nil {
					fmt.Printf("Error fetching logs: %v\n", err)
				}
				preview := string(output)
				if len(preview) > 2000 {
					preview = preview[len(preview)-2000:]
				}
				fmt.Println(preview)
				fmt.Print("\nProceed with AI analysis of these logs? (y/n): ")
				confirm, _ := reader.ReadString('\n')
				confirm = strings.TrimSpace(strings.ToLower(confirm))
				if confirm != "y" && confirm != "yes" {
					fmt.Println("Log analysis cancelled.")
					continue
				}
				// AI or rule-based analysis
				var analysis string
				if claudeConfig != nil {
					prompt := `You are a DevOps assistant. Analyze the following logs for errors, warnings, or issues. If you find problems, explain them, suggest a fix, and provide a command to resolve if possible. If all looks fine, say so.\n\nLOGS:\n` + preview
					analysis = callClaude(claudeConfig, "Log Analysis", prompt)
				} else {
					analysis = simpleLogAnalysis(preview)
				}
				fmt.Println("\n--- AI Log Analysis ---")
				fmt.Println(analysis)
				continue
			}
			// Show operation details and prompt for confirmation
			fmt.Printf("\n💡 Operation: %s\nCommand: %s\nDescription: %s\n", suggestion.Intent, suggestion.Command, suggestion.Description)
			fmt.Print("Would you like to execute this operation? (y/n): ")
			confirm, _ := reader.ReadString('\n')
			confirm = strings.TrimSpace(strings.ToLower(confirm))
			if confirm == "y" || confirm == "yes" {
				go executeCommand(suggestion)
			} else {
				fmt.Println("❌ Operation cancelled.")
			}
		} else {
			fmt.Println("❌ Could not understand the operation.")
		}
	}
}

func runKafkaAdminSession(brokers string, commandConfig string) {
	// Prerequisite check for Homebrew on macOS
	if runtime.GOOS == "darwin" {
		if _, err := findCommand("brew"); err != nil {
			fmt.Println(yellow + "⚠️  Homebrew is not available in your PATH." + reset)
			fmt.Println("   ops0 uses Homebrew to manage software on macOS. To proceed, you must")
			fmt.Println("   ensure Homebrew is installed and configured correctly.")
			fmt.Println("\n   Please run the appropriate command for your system, then " + bold + "restart your terminal" + reset + ":")

			// Suggest command based on architecture
			var brewPath string
			if runtime.GOARCH == "arm64" { // Apple Silicon
				brewPath = "/opt/homebrew/bin"
			} else { // Intel
				brewPath = "/usr/local/bin"
			}

			shell := os.Getenv("SHELL")
			profileFile := "~/.zshrc" // default for modern macOS
			if strings.Contains(shell, "bash") {
				profileFile = "~/.bash_profile"
			}

			fmt.Printf(bold+"   echo 'export PATH=\"%s:$PATH\"' >> %s"+reset+"\n\n", brewPath, profileFile)
			fmt.Println("   If you don't have Homebrew installed, visit https://brew.sh")
			os.Exit(1)
		}
	}

	// 1. Check if kafka-topics is available
	cmdPath, err := findCommand("kafka-topics")
	if err != nil {
		if err.Error() == "found_not_in_path" {
			fmt.Println(yellow + "⚠️  Kafka tools are installed but not found in your current PATH." + reset)
			fmt.Println("   This is common after installing with Homebrew. To fix this,")
			fmt.Println("   add Homebrew's bin directory to your shell's configuration file.")

			// Suggest command based on shell
			shell := os.Getenv("SHELL")
			profileFile := "~/.bash_profile" // default
			if strings.Contains(shell, "zsh") {
				profileFile = "~/.zshrc"
			} else if strings.Contains(shell, "bash") {
				profileFile = "~/.bashrc"
			}

			brewPath := filepath.Dir(cmdPath)

			fmt.Println("\n   Run this command, then " + bold + "restart your terminal" + reset + ":")
			fmt.Printf(bold+"   echo 'export PATH=\"%s:$PATH\"' >> %s"+reset+"\n\n", brewPath, profileFile)
		} else { // "not_found"
			fmt.Println(red + "❌ Kafka command-line tools not found." + reset)
			fmt.Print("Would you like to try and install Kafka now? (y/n): ")
			if getUserConfirmation() {
				kafkaTool := &Tool{
					Name:       "kafka",
					CheckCmd:   "kafka-topics --version",
					InstallCmd: getInstallCommand("kafka"),
				}
				if installTool(kafkaTool) {
					fmt.Println(green + "✅ Kafka installed successfully!" + reset)
					fmt.Println(yellow + "Please " + bold + "restart your terminal session" + reset + " for the PATH changes to take effect, then run the command again." + reset)
				} else {
					fmt.Println(red + "❌ Kafka installation failed. Please install it manually." + reset)
				}
			} else {
				fmt.Println("   Exiting. Please install Kafka and ensure its 'bin' directory is in your system's PATH.")
			}
		}
		os.Exit(1)
	}

	// 2. Test connection to the cluster
	fmt.Printf("Connecting to Kafka cluster at %s...\n", brokers)
	args := []string{"--bootstrap-server", brokers, "--list"}
	if commandConfig != "" {
		args = append(args, "--command-config", commandConfig)
	}
	testCmd := exec.Command(cmdPath, args...)
	testCmd.Stderr = os.Stderr
	if err := testCmd.Run(); err != nil {
		fmt.Printf(red+"❌ Could not connect to Kafka cluster. Please check your broker addresses and network connectivity. Error: %v"+reset+"\n", err)
		os.Exit(1)
	}
	fmt.Println(green + "✅ Connection successful." + reset)

	// 3. Setup interactive session
	fmt.Printf("Entering Kafka Admin Mode. Type 'quit' or 'exit' to leave, or 'stats' to see session statistics.\n")
	reader := bufio.NewReader(os.Stdin)
	claudeConfig := getClaudeConfigIfAvailable()
	if claudeConfig == nil {
		fmt.Println(yellow + "⚠️  Warning: ANTHROPIC_API_KEY not set. Kafka admin mode requires AI." + reset)
		fmt.Println("   Please set the key to enable natural language commands.")
		os.Exit(1)
	}
	kafkaStats := make(map[string]int)

	// 4. Start REPL
	for {
		fmt.Printf(blue+"kafka-admin@%s> "+reset, brokers)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "quit" || input == "exit" {
			fmt.Println("👋 Exiting Kafka Admin Mode.")
			break
		}
		if input == "stats" {
			displayKafkaStats(kafkaStats)
			continue
		}
		if input == "" {
			continue
		}

		suggestion := getKafkaAISuggestion(claudeConfig, input, brokers, commandConfig)

		if suggestion != nil {
			// Show operation details and prompt for confirmation
			fmt.Printf("\n"+bold+"💡 Suggested Operation:"+reset+"\n")
			fmt.Printf("   Intent: %s\n", suggestion.Intent)
			fmt.Printf("   Command: %s\n", suggestion.Command)
			fmt.Printf("   Description: %s\n", suggestion.Description)
			fmt.Print("\nExecute this operation? (y/n): ")

			confirm, _ := reader.ReadString('\n')
			confirm = strings.TrimSpace(strings.ToLower(confirm))

			if confirm == "y" || confirm == "yes" {
				// Prepend full path to the executable part of the command string
				parts := strings.Fields(suggestion.Command)
				if len(parts) > 0 && !strings.Contains(parts[0], "/") {
					baseCmd := parts[0]
					// Find the full path to the base command
					fullCmdPath, findErr := findCommand(baseCmd)
					if findErr == nil {
						suggestion.Command = strings.Replace(suggestion.Command, baseCmd, fullCmdPath, 1)
					}
				}
				if suggestion.Intent != "" {
					kafkaStats[suggestion.Intent]++
				}
				executeCommand(suggestion)
			} else {
				fmt.Println("❌ Operation cancelled.")
			}
		} else {
			fmt.Println("❌ Could not understand the Kafka operation.")
		}
	}
}

func getKafkaAISuggestion(config *ClaudeConfig, userInput, brokers, commandConfig string) *CommandSuggestion {
	connectionFlags := fmt.Sprintf("--bootstrap-server %s", brokers)
	if commandConfig != "" {
		connectionFlags += fmt.Sprintf(" --command-config %s", commandConfig)
	}

	systemPrompt := fmt.Sprintf(`You are an expert Kafka administrator's assistant. Your sole job is to translate natural language user requests into the appropriate Kafka command-line tool command (e.g., kafka-topics, kafka-console-consumer, kafka-configs).

The user is connected to a Kafka cluster.
**You must inject '%s' into every command you generate.** Do not use full paths for the kafka commands (e.g. use 'kafka-topics' not '/usr/local/bin/kafka-topics').

Here are some examples of Kafka commands:
- List topics: kafka-topics %s --list
- Describe a topic: kafka-topics %s --describe --topic my-topic
- Create a topic: kafka-topics %s --create --topic new-topic --partitions 1 --replication-factor 1
- Delete a topic: kafka-topics %s --delete --topic old-topic
- Consume messages: kafka-console-consumer %s --topic my-topic --from-beginning --max-messages 10
- Produce a message: kafka-console-producer %s --topic my-topic
- Describe configs: kafka-configs %s --describe --entity-type topics --entity-name my-topic

Respond with a JSON object in this exact format, with no extra text or explanations.
Use one of the following standardized intents: 'list_topics', 'describe_topic', 'create_topic', 'delete_topic', 'produce_message', 'consume_message', 'alter_configs', 'describe_configs', 'list_consumer_groups', 'describe_consumer_group', 'get_cluster_info'.
{
  "tool": "kafka",
  "command": "kafka-topics %s --list",
  "dry_run_command": "",
  "description": "This command will list all topics in the Kafka cluster.",
  "intent": "list_topics",
  "confidence": 0.98,
  "has_dry_run": false
}

If the user says "produce a message 'hello world' to topic 'test'", the command should be:
"echo 'hello world' | kafka-console-producer %s --topic test"

User Request: %s`, connectionFlags, connectionFlags, connectionFlags, connectionFlags, connectionFlags, connectionFlags, connectionFlags, connectionFlags, connectionFlags, connectionFlags, userInput)

	response := callClaude(config, systemPrompt, userInput)
	if response == "" {
		return nil
	}

	var suggestion CommandSuggestion
	if err := json.Unmarshal([]byte(response), &suggestion); err != nil {
		fmt.Printf("⚠️  ops0: AI response parsing error: %v\n", err)
		return nil
	}

	suggestion.AIGenerated = true
	suggestion.Tool = "kafka" // Ensure tool is always set to kafka
	return &suggestion
}

func displayKafkaStats(stats map[string]int) {
	fmt.Println("\n📊 Kafka Admin Session Stats")
	fmt.Println("══════════════════════════════")
	if len(stats) == 0 {
		fmt.Println("No operations performed yet in this session.")
		fmt.Println()
		return
	}

	total := 0
	for _, count := range stats {
		total += count
	}
	fmt.Printf("Total Operations: %d\n", total)
	fmt.Println("Operation Breakdown:")

	// Sort keys for consistent order
	keys := make([]string, 0, len(stats))
	for k := range stats {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, op := range keys {
		fmt.Printf("  - %s: %d\n", op, stats[op])
	}
	fmt.Println()
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
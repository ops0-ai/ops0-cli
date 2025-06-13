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

func main() {
	// Handle flags
	var showVersion bool
	var displayHelp bool
	var message string
	var aiMode bool
	var troubleshoot bool
	var showStats bool
	
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.BoolVar(&displayHelp, "help", false, "show help information")
	flag.StringVar(&message, "m", "", "natural language command message")
	flag.BoolVar(&aiMode, "ai", false, "use AI mode for advanced command generation")
	flag.BoolVar(&troubleshoot, "troubleshoot", false, "troubleshooting mode with context analysis")
	flag.BoolVar(&showStats, "stats", false, "show usage statistics")
	flag.Parse()

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
		suggestion = parseIntent(message)
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
	fmt.Println("  ops0 -m \"command\" -ai")
	fmt.Println("  ops0 -m \"error description\" -troubleshoot")
	fmt.Println("  ops0 -version")
	fmt.Println("  ops0 -help")

	// Flags
	fmt.Println("\n🚩 Flags:")
	fmt.Println("  -m           Natural language command message (required)")
	fmt.Println("  -ai          Enable AI mode for advanced command generation")
	fmt.Println("  -troubleshoot Enable troubleshooting mode with context analysis")
	fmt.Println("  -version     Show version information")
	fmt.Println("  -help        Show this help message")

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

func parseIntent(input string) *CommandSuggestion {
	input = strings.ToLower(input)
	
	// System Admin patterns - Check these first
	if matched, _ := regexp.MatchString(`(check|show|display|get).*(disk|memory|cpu|system).*usage|df.*-h|free.*-h|top`, input); matched {
		// Check if it's specifically about local system
		if strings.Contains(input, "device") || strings.Contains(input, "local") || 
		   strings.Contains(input, "my") || strings.Contains(input, "machine") || 
		   strings.Contains(input, "system") {
			return &CommandSuggestion{
				Tool:        "system_admin",
				Command:     extractSystemMonitorCommand(input),
				Description: "This will show system resource usage and monitoring information for your local machine.",
				Intent:      "monitor local system resources",
				Confidence:  0.95,
				AIGenerated: false,
				HasDryRun:   false,
			}
		}
	}

	// Terraform patterns
	if matched, _ := regexp.MatchString(`(plan|planning).*iac|terraform.*plan|infrastructure.*plan`, input); matched {
		return &CommandSuggestion{
			Tool:        "terraform",
			Command:     "terraform plan",
			DryRunCommand: "", // Plan is already a dry run
			Description: "This will show you what changes Terraform will make to your infrastructure without actually applying them.",
			Intent:      "plan infrastructure changes",
			Confidence:  0.8,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}
	
	if matched, _ := regexp.MatchString(`apply.*terraform|deploy.*infrastructure|apply.*iac`, input); matched {
		return &CommandSuggestion{
			Tool:        "terraform",
			Command:     "terraform apply",
			DryRunCommand: "terraform plan",
			Description: "This will apply your Terraform configuration and make the actual infrastructure changes.",
			Intent:      "apply infrastructure changes",
			Confidence:  0.8,
			AIGenerated: false,
			HasDryRun:   true,
		}
	}
	
	if matched, _ := regexp.MatchString(`destroy.*infrastructure|tear.*down|terraform.*destroy`, input); matched {
		return &CommandSuggestion{
			Tool:        "terraform",
			Command:     "terraform destroy",
			DryRunCommand: "terraform plan -destroy",
			Description: "This will destroy all resources managed by your Terraform configuration.",
			Intent:      "destroy infrastructure",
			Confidence:  0.8,
			AIGenerated: false,
			HasDryRun:   true,
		}
	}

	// Ansible patterns
	if matched, _ := regexp.MatchString(`(setup|create|init|generate).*ansible.*project|ansible.*playbook.*inventory`, input); matched {
		return &CommandSuggestion{
			Tool:    "ansible_scaffold",
			Command: input, // Pass the full user message for AI/template
			Description: "Scaffold a new Ansible project (playbook + inventory) from your request.",
			Intent:  "scaffold ansible project",
			Confidence: 0.95,
			AIGenerated: false,
			HasDryRun: false,
		}
	}
	
	if matched, _ := regexp.MatchString(`run.*playbook|execute.*ansible|ansible.*playbook`, input); matched {
		return &CommandSuggestion{
			Tool:        "ansible",
			Command:     "ansible-playbook playbook.yml",
			DryRunCommand: "ansible-playbook playbook.yml --check",
			Description: "This will run your Ansible playbook to configure your servers.",
			Intent:      "run ansible playbook",
			Confidence:  0.8,
			AIGenerated: false,
			HasDryRun:   true,
		}
	}
	
	if matched, _ := regexp.MatchString(`check.*ansible|dry.*run.*ansible|ansible.*check`, input); matched {
		return &CommandSuggestion{
			Tool:        "ansible",
			Command:     "ansible-playbook playbook.yml --check",
			Description: "This will do a dry run of your Ansible playbook without making actual changes.",
			Intent:      "check ansible playbook",
			Confidence:  0.8,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}

	// Kubernetes patterns
	if matched, _ := regexp.MatchString(`(get|list|show).*pods?|pods?.*status|check.*pods?`, input); matched {
		// Check if namespace is specified
		if strings.Contains(input, "namespace") {
			// Extract namespace if possible
			words := strings.Fields(input)
			for i, word := range words {
				if word == "namespace" && i+1 < len(words) {
					namespace := words[i+1]
					return &CommandSuggestion{
						Tool:        "kubectl",
						Command:     "kubectl get pods -n " + namespace,
						Description: "This will show all pods in the " + namespace + " namespace and their status.",
						Intent:      "check pod status in specific namespace",
						Confidence:  0.9,
						AIGenerated: false,
						HasDryRun:   false,
					}
				}
			}
		}
		return &CommandSuggestion{
			Tool:        "kubectl",
			Command:     "kubectl get pods",
			Description: "This will show all pods in the current namespace and their status.",
			Intent:      "check pod status",
			Confidence:  0.8,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}
	
	if matched, _ := regexp.MatchString(`apply.*kubernetes|deploy.*k8s|kubectl.*apply`, input); matched {
		return &CommandSuggestion{
			Tool:        "kubectl",
			Command:     "kubectl apply -f .",
			DryRunCommand: "kubectl apply -f . --dry-run=client",
			Description: "This will apply Kubernetes manifests in the current directory.",
			Intent:      "apply kubernetes manifests",
			Confidence:  0.8,
			AIGenerated: false,
			HasDryRun:   true,
		}
	}
	
	if matched, _ := regexp.MatchString(`delete.*kubernetes|remove.*k8s|kubectl.*delete`, input); matched {
		return &CommandSuggestion{
			Tool:        "kubectl",
			Command:     "kubectl delete -f .",
			DryRunCommand: "kubectl delete -f . --dry-run=client",
			Description: "This will delete resources defined in Kubernetes manifests in the current directory.",
			Intent:      "delete kubernetes resources",
			Confidence:  0.8,
			AIGenerated: false,
			HasDryRun:   true,
		}
	}

	// Docker patterns
	if matched, _ := regexp.MatchString(`(list|show|get).*containers?|containers?.*running|ps`, input); matched {
		return &CommandSuggestion{
			Tool:        "docker",
			Command:     "docker ps",
			Description: "This will show all currently running Docker containers.",
			Intent:      "list running containers",
			Confidence:  0.9,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}
	
	if matched, _ := regexp.MatchString(`build.*image|docker.*build`, input); matched {
		return &CommandSuggestion{
			Tool:        "docker",
			Command:     "docker build -t my-app .",
			Description: "This will build a Docker image from the Dockerfile in current directory.",
			Intent:      "build docker image",
			Confidence:  0.8,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}
	
	if matched, _ := regexp.MatchString(`(list|show|get).*images?|images?.*list`, input); matched {
		return &CommandSuggestion{
			Tool:        "docker",
			Command:     "docker images",
			Description: "This will show all Docker images on your system.",
			Intent:      "list docker images",
			Confidence:  0.9,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}

	// AWS CLI patterns
	if matched, _ := regexp.MatchString(`(list|show|get).*ec2|instances?.*list|ec2.*instances?`, input); matched {
		return &CommandSuggestion{
			Tool:        "aws",
			Command:     "aws ec2 describe-instances",
			Description: "This will show all EC2 instances in your AWS account.",
			Intent:      "list EC2 instances",
			Confidence:  0.8,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}
	
	if matched, _ := regexp.MatchString(`(list|show|get).*s3|buckets?.*list|s3.*buckets?`, input); matched {
		return &CommandSuggestion{
			Tool:        "aws",
			Command:     "aws s3 ls",
			Description: "This will list all S3 buckets in your AWS account.",
			Intent:      "list S3 buckets",
			Confidence:  0.8,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}

	// System Admin patterns
	if matched, _ := regexp.MatchString(`(install|update|upgrade).*package|apt.*install|yum.*install|dnf.*install`, input); matched {
		return &CommandSuggestion{
			Tool:        "system_admin",
			Command:     extractPackageCommand(input),
			Description: "This will manage packages on your system using the appropriate package manager.",
			Intent:      "manage system packages",
			Confidence:  0.9,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}
	
	if matched, _ := regexp.MatchString(`(start|stop|restart|status).*service|systemctl.*service`, input); matched {
		return &CommandSuggestion{
			Tool:        "system_admin",
			Command:     extractServiceCommand(input),
			Description: "This will manage system services using systemctl.",
			Intent:      "manage system services",
			Confidence:  0.9,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}
	
	if matched, _ := regexp.MatchString(`(check|show|display).*logs|journalctl|tail.*log`, input); matched {
		return &CommandSuggestion{
			Tool:        "system_admin",
			Command:     extractLogCommand(input),
			Description: "This will show system logs and journal entries.",
			Intent:      "view system logs",
			Confidence:  0.9,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}

	// Log analysis patterns
	if matched, _ := regexp.MatchString(`(analyze|review|check|summarize|inspect).*(logs?|log files?|pod logs?)`, input); matched {
		// Kubernetes pod log analysis
		podRe := regexp.MustCompile(`pod\s+([a-zA-Z0-9-]+)`)
		nsRe := regexp.MustCompile(`namespace\s+([a-zA-Z0-9-]+)`)
		pod := ""
		ns := "default"
		if m := podRe.FindStringSubmatch(input); len(m) > 1 {
			pod = m[1]
		}
		if m := nsRe.FindStringSubmatch(input); len(m) > 1 {
			ns = m[1]
		}
		if pod != "" {
			return &CommandSuggestion{
				Tool:        "kubectl",
				Command:     "kubectl logs " + pod + " -n " + ns + " --tail=100",
				Description: "Fetch and analyze the last 100 log lines for pod '" + pod + "' in namespace '" + ns + "'.",
				Intent:      "analyze_logs",
				Confidence:  0.95,
				AIGenerated: false,
				HasDryRun:   false,
			}
		}
		// Fallback: generic log analysis
		return &CommandSuggestion{
			Tool:        "system_admin",
			Command:     extractLogCommand(input),
			Description: "Fetch and analyze recent system logs.",
			Intent:      "analyze_logs",
			Confidence:  0.9,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}

	// Log file analysis pattern
	fileRe := regexp.MustCompile(`(?:analyze|review|check|summarize|inspect)[^\n]*?(/[^\s]+\.log)`) // non-greedy match for file path
	if m := fileRe.FindStringSubmatch(input); len(m) > 1 {
		filePath := m[1]
		return &CommandSuggestion{
			Tool:        "system_admin",
			Command:     "tail -n 100 " + filePath,
			Description: "Fetch and analyze the last 100 lines of log file: " + filePath,
			Intent:      "analyze_logs",
			Confidence:  0.95,
			AIGenerated: false,
			HasDryRun:   false,
		}
	}

	return nil
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

func handleInteraction(suggestion *CommandSuggestion) {
	// Handle log analysis intent for any tool
	if suggestion.Intent == "analyze_logs" {
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
		if !getUserConfirmation() {
			fmt.Println("Log analysis cancelled.")
			return
		}
		// AI or rule-based analysis
		var analysis string
		if claudeConfig := getClaudeConfigIfAvailable(); claudeConfig != nil {
			prompt := `You are a DevOps assistant. Analyze the following logs for errors, warnings, or issues. If you find problems, explain them, suggest a fix, and provide a command to resolve if possible. If all looks fine, say so.\n\nLOGS:\n` + preview
			analysis = callClaude(claudeConfig, "Log Analysis", prompt)
		} else {
			analysis = simpleLogAnalysis(preview)
		}
		fmt.Println("\n--- AI Log Analysis ---")
		fmt.Println(analysis)
		return
	}

	// Normalize tool name for installation
	toolName := suggestion.Tool
	if toolName == "aws-cli" {
		toolName = "aws"
	}
	
	// Skip installation check for system_admin as it uses built-in commands
	if toolName == "system_admin" {
		// Prepare command details for display
		var details []string
		details = append(details, "Tool: System Administration")
		details = append(details, "Intent: "+suggestion.Intent)
		details = append(details, "Command: "+suggestion.Command)
		if suggestion.HasDryRun {
			details = append(details, "Dry Run: "+suggestion.DryRunCommand)
		}
		details = append(details, "Description: "+suggestion.Description)
		if suggestion.AIGenerated {
			details = append(details, fmt.Sprintf("AI Confidence: %.0f%%", suggestion.Confidence*100))
		}

		// Display command details
		if suggestion.AIGenerated {
			fmt.Print(formatSection("🧠 AI-Generated Command", details))
		} else {
			fmt.Print(formatSection("💡 Command Details", details))
		}

		fmt.Print("\nWould you like to execute this command? (y/n): ")
		if !getUserConfirmation() {
			fmt.Print("\n👋 No problem! Let me know if you need help with anything else.\n")
			return
		}

		executeCommand(suggestion)
		return
	}
	
	tool := &Tool{
		Name:       toolName,
		CheckCmd:   toolName + " --version",
		InstallCmd: getInstallCommand(toolName),
	}
	
	tool.IsInstalled = checkToolInstalled(tool)
	
	// Prepare command details for display
	var details []string
	details = append(details, "Tool: "+getToolDisplayName(suggestion.Tool))
	details = append(details, "Intent: "+suggestion.Intent)
	details = append(details, "Command: "+suggestion.Command)
	if suggestion.HasDryRun {
		details = append(details, "Dry Run: "+suggestion.DryRunCommand)
	}
	details = append(details, "Description: "+suggestion.Description)
	if suggestion.AIGenerated {
		details = append(details, fmt.Sprintf("AI Confidence: %.0f%%", suggestion.Confidence*100))
	}

	// Display command details
	if suggestion.AIGenerated {
		fmt.Print(formatSection("🧠 AI-Generated Command", details))
	} else {
		fmt.Print(formatSection("💡 Command Details", details))
	}

	// Check if tool is installed
	if !tool.IsInstalled {
		toolDisplayName := getToolDisplayName(suggestion.Tool)
		fmt.Printf("\n" + yellow + bold + "⚠️  Installation Required" + reset + "\n")
		fmt.Printf("%s is not installed on your system.\n", toolDisplayName)
		fmt.Print("Would you like to install it? (y/n): ")
		
		if getUserConfirmation() {
			if installTool(tool) {
				fmt.Printf("\n" + green + "✅ %s installed successfully!" + reset + "\n", toolDisplayName)
			} else {
				fmt.Printf("\n" + red + "❌ Failed to install %s. Please install it manually." + reset + "\n", toolDisplayName)
				return
			}
		} else {
			fmt.Printf("\n" + red + "❌ Cannot proceed without %s. Please install it and try again." + reset + "\n", toolDisplayName)
			return
		}
	}

	// Handle dry run option
	if suggestion.HasDryRun {
		fmt.Print("\n" + bold + "🔍 Dry Run Available" + reset + "\n")
		fmt.Print("Would you like to perform a dry run first? (y/n): ")
		if getUserConfirmation() {
			fmt.Printf("\n" + bold + "🔍 Performing dry run..." + reset + "\n")
			executeDryRun(suggestion)
			fmt.Print("\nWould you like to proceed with the actual command? (y/n): ")
			if !getUserConfirmation() {
				fmt.Print("\n👋 No problem! Let me know if you need help with anything else.\n")
				return
			}
		}
	} else {
		fmt.Print("\nWould you like to execute this command? (y/n): ")
		if !getUserConfirmation() {
			fmt.Print("\n👋 No problem! Let me know if you need help with anything else.\n")
			return
		}
	}

	if suggestion.Tool == "ansible_scaffold" {
		var files map[string]string
		var err error
		intent := strings.ToLower(suggestion.Intent + " " + suggestion.Command)
		projectName := extractProjectName(suggestion.Command)
		if projectName == "" {
			projectName = "ansible_project"
		}
		dir := projectName
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			files, err = parseAnsibleFilesFromAIDescription(suggestion.Description)
			if err != nil || len(files) == 0 {
				// fallback to previous AI parsing if needed
				var playbookContent, inventoryContent, playbookFile, inventoryFile string
				playbookContent, inventoryContent, playbookFile, inventoryFile, err = generateAnsibleProjectAIWithFilenames(suggestion.Command)
				if err == nil {
					files = map[string]string{
						playbookFile: playbookContent,
						inventoryFile: inventoryContent,
					}
				}
			}
		} else {
			var playbookContent, inventoryContent string
			playbookContent, inventoryContent, err = generateAnsibleProjectTemplate(suggestion.Command)
			files = map[string]string{
				"playbook.yml": playbookContent,
				"inventory.yml": inventoryContent,
			}
		}
		if err != nil || len(files) == 0 {
			fmt.Printf("❌ Failed to generate Ansible project: %v\n", err)
			return
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("❌ Could not create project directory '%s': %v\n", dir, err)
			return
		}
		fmt.Printf("✅ Ansible project directory '%s' created with:\n", dir)
		for fname, content := range files {
			fpath := dir + "/" + fname
			if err := os.WriteFile(fpath, []byte(content), 0644); err != nil {
				fmt.Printf("❌ Could not write %s: %v\n", fpath, err)
				return
			}
			fmt.Printf("  - %s\n", fname)
		}
		// Only execute if the user intent is to run/execute, not create/setup/init/generate
		if strings.Contains(intent, "run") || strings.Contains(intent, "execute") || strings.Contains(intent, "do ") {
			playbookFile, inventoryFile := findAnsiblePlaybookAndInventory(files)
			if playbookFile != "" && inventoryFile != "" {
				fmt.Print("\nWould you like to execute the playbook now? (y/n): ")
				if getUserConfirmation() {
					cmd := exec.Command("ansible-playbook", "-i", inventoryFile, playbookFile)
					cmd.Dir = dir
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					cmd.Stdin = os.Stdin
					fmt.Printf("\n🚀 Executing: ansible-playbook -i %s %s in %s\n\n", inventoryFile, playbookFile, dir)
					if err := cmd.Run(); err != nil {
						fmt.Printf("\n❌ Command failed with error: %v\n", err)
					} else {
						fmt.Printf("\n✅ Playbook executed successfully!\n")
					}
				} else {
					fmt.Println("\n👋 Project is ready. You can run the playbook later with:\n  cd", dir, "&& ansible-playbook -i", inventoryFile, playbookFile)
				}
			} else {
				fmt.Println("\n⚠️  Could not determine playbook/inventory file for execution. Please check the generated files.")
			}
		} else {
			fmt.Println("\n👋 Project is ready. You can run the playbook later with:\n  cd", dir, "&& ansible-playbook -i inventory.yml playbook.yml")
		}
		return
	}

	executeCommand(suggestion)
}

func executeCommand(suggestion *CommandSuggestion) {
	fmt.Printf("\n" + bold + "🚀 Executing: " + reset + "%s\n\n", suggestion.Command)
	
	command := suggestion.Command
	if suggestion.Tool == "ansible" && strings.Contains(command, "playbook.yml") {
		if playbookFile := findPlaybookFile(); playbookFile != "" {
			command = strings.Replace(command, "playbook.yml", playbookFile, 1)
			fmt.Printf(bold + "📝 Found playbook: " + reset + "%s\n", playbookFile)
		}
	}
	
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("\n" + red + "❌ Command failed with error: %v" + reset + "\n", err)
	} else {
		fmt.Printf("\n" + green + "✅ Command completed successfully!" + reset + "\n")
	}

	// Log command usage
	logCommandStat(suggestion.Tool, command)
}

func executeDryRun(suggestion *CommandSuggestion) {
	if suggestion.DryRunCommand == "" {
		return
	}

	fmt.Printf(bold + "🔍 Executing dry run: " + reset + "%s\n\n", suggestion.DryRunCommand)
	
	cmd := exec.Command("sh", "-c", suggestion.DryRunCommand)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("\n" + yellow + "⚠️  Dry run completed with warnings/errors: %v" + reset + "\n", err)
	} else {
		fmt.Printf("\n" + green + "✅ Dry run completed successfully!" + reset + "\n")
	}
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

type cmdCount struct {
	cmd   string
	count int
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
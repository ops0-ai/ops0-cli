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
	"runtime"
	"strings"
	"regexp"
	"flag"
	"time"
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
	Description string
	Intent      string
	Confidence  float64
	AIGenerated bool
}

func main() {
	// Handle flags
	var showVersion bool
	var message string
	var aiMode bool
	var troubleshoot bool
	
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.StringVar(&message, "m", "", "natural language command message")
	flag.BoolVar(&aiMode, "ai", false, "use AI mode for advanced command generation")
	flag.BoolVar(&troubleshoot, "troubleshoot", false, "troubleshooting mode with context analysis")
	flag.Parse()

	if showVersion {
		fmt.Printf("ops0 version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("built: %s\n", date)
		fmt.Printf("go version: %s\n", runtime.Version())
		fmt.Printf("platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return
	}

	// Check if message was provided
	if message == "" {
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
	fmt.Println("Usage:")
	fmt.Println("  ops0 -m \"your natural language command\"")
	fmt.Println("  ops0 -m \"command\" -ai")
	fmt.Println("  ops0 -m \"error description\" -troubleshoot")
	fmt.Println("  ops0 -version")
	fmt.Println("\nExamples:")
	fmt.Println("  ops0 -m \"i want to plan my iac code\"")
	fmt.Println("  ops0 -m \"show running containers\"")
	fmt.Println("  ops0 -m \"check if my pods are running\" -ai")
}

func parseIntent(input string) *CommandSuggestion {
	input = strings.ToLower(input)
	
	// Terraform patterns
	if matched, _ := regexp.MatchString(`(plan|planning).*iac|terraform.*plan|infrastructure.*plan`, input); matched {
		return &CommandSuggestion{
			Tool:        "terraform",
			Command:     "terraform plan",
			Description: "This will show you what changes Terraform will make to your infrastructure without actually applying them.",
			Intent:      "plan infrastructure changes",
			Confidence:  0.8,
			AIGenerated: false,
		}
	}
	
	if matched, _ := regexp.MatchString(`apply.*terraform|deploy.*infrastructure|apply.*iac`, input); matched {
		return &CommandSuggestion{
			Tool:        "terraform",
			Command:     "terraform apply",
			Description: "This will apply your Terraform configuration and make the actual infrastructure changes.",
			Intent:      "apply infrastructure changes",
			Confidence:  0.8,
			AIGenerated: false,
		}
	}
	
	if matched, _ := regexp.MatchString(`destroy.*infrastructure|tear.*down|terraform.*destroy`, input); matched {
		return &CommandSuggestion{
			Tool:        "terraform",
			Command:     "terraform destroy",
			Description: "This will destroy all resources managed by your Terraform configuration.",
			Intent:      "destroy infrastructure",
			Confidence:  0.8,
			AIGenerated: false,
		}
	}

	// Ansible patterns
	if matched, _ := regexp.MatchString(`run.*playbook|execute.*ansible|ansible.*playbook`, input); matched {
		return &CommandSuggestion{
			Tool:        "ansible",
			Command:     "ansible-playbook playbook.yml",
			Description: "This will run your Ansible playbook to configure your servers.",
			Intent:      "run ansible playbook",
			Confidence:  0.8,
			AIGenerated: false,
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
		}
	}
	
	if matched, _ := regexp.MatchString(`(get|list|show).*services?|services?.*status`, input); matched {
		// Check if namespace is specified
		if strings.Contains(input, "namespace") {
			words := strings.Fields(input)
			for i, word := range words {
				if word == "namespace" && i+1 < len(words) {
					namespace := words[i+1]
					return &CommandSuggestion{
						Tool:        "kubectl",
						Command:     "kubectl get services -n " + namespace,
						Description: "This will show all services in the " + namespace + " namespace.",
						Intent:      "check service status in specific namespace",
						Confidence:  0.9,
						AIGenerated: false,
					}
				}
			}
		}
		return &CommandSuggestion{
			Tool:        "kubectl",
			Command:     "kubectl get services",
			Description: "This will show all services in the current namespace.",
			Intent:      "check service status",
			Confidence:  0.8,
			AIGenerated: false,
		}
	}
	
	if matched, _ := regexp.MatchString(`logs?|check.*logs?|pod.*logs?`, input); matched {
		return &CommandSuggestion{
			Tool:        "kubectl",
			Command:     "kubectl logs -l app=<app-name>",
			Description: "This will show logs from pods. Replace <app-name> with your app label.",
			Intent:      "view pod logs",
			Confidence:  0.7,
			AIGenerated: false,
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
		}
	}

	return nil
}

func getAISuggestion(config *ClaudeConfig, userInput string) *CommandSuggestion {
	systemPrompt := `You are ops0, an AI-powered DevOps CLI assistant. Your job is to translate natural language requests into specific DevOps commands.

You support these tools: terraform, ansible, kubectl, docker, helm, aws-cli, gcloud, azure-cli.

Respond with a JSON object in this exact format:
{
  "tool": "terraform",
  "command": "terraform plan",
  "description": "This will show you what changes Terraform will make",
  "intent": "plan infrastructure changes",
  "confidence": 0.95
}

Rules:
- Only suggest commands for tools that are commonly available
- Prefer safe, read-only commands when possible
- Include helpful descriptions
- Set confidence between 0-1 based on how certain you are
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

func handleInteraction(suggestion *CommandSuggestion) {
	tool := &Tool{
		Name:       suggestion.Tool,
		CheckCmd:   suggestion.Tool + " --version",
		InstallCmd: getInstallCommand(suggestion.Tool),
	}
	
	tool.IsInstalled = checkToolInstalled(tool)
	
	aiIndicator := ""
	if suggestion.AIGenerated {
		aiIndicator = "🧠 "
		fmt.Printf("💡 %sops0: I can help you %s via `%s`. %s (Confidence: %.0f%%)\n", 
			aiIndicator, suggestion.Intent, suggestion.Command, suggestion.Description, suggestion.Confidence*100)
	} else {
		fmt.Printf("💡 ops0: I can help you %s via `%s`. %s\n", 
			suggestion.Intent, suggestion.Command, suggestion.Description)
	}
	
	fmt.Print("Would you like me to do this? (y/n): ")
	
	if !getUserConfirmation() {
		fmt.Println("👋 ops0: No problem! Let me know if you need help with anything else.")
		return
	}

	if !tool.IsInstalled {
		fmt.Printf("⚠️  ops0: %s is not installed on your system.\n", strings.Title(tool.Name))
		fmt.Print("Would you like me to install it? (y/n): ")
		
		if getUserConfirmation() {
			if installTool(tool) {
				fmt.Printf("✅ ops0: %s installed successfully!\n", strings.Title(tool.Name))
			} else {
				fmt.Printf("❌ ops0: Failed to install %s. Please install it manually.\n", strings.Title(tool.Name))
				return
			}
		} else {
			fmt.Printf("❌ ops0: Cannot proceed without %s. Please install it and try again.\n", strings.Title(tool.Name))
			return
		}
	}

	executeCommand(suggestion)
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
	if runtime.GOOS != "darwin" {
		return ""
	}
	
	switch toolName {
	case "terraform":
		return "brew install terraform"
	case "ansible":
		return "brew install ansible"
	case "kubectl":
		return "brew install kubectl"
	case "docker":
		return "brew install docker"
	case "helm":
		return "brew install helm"
	case "aws":
		return "brew install awscli"
	case "gcloud":
		return "brew install google-cloud-sdk"
	case "az":
		return "brew install azure-cli"
	default:
		return ""
	}
}

func installTool(tool *Tool) bool {
	if tool.InstallCmd == "" {
		fmt.Printf("❌ ops0: Don't know how to install %s on this system.\n", tool.Name)
		return false
	}
	
	fmt.Printf("🔧 ops0: Installing %s...\n", tool.Name)
	cmd := exec.Command("sh", "-c", tool.InstallCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run() == nil
}

func executeCommand(suggestion *CommandSuggestion) {
	fmt.Printf("🚀 ops0: Executing: %s\n\n", suggestion.Command)
	
	command := suggestion.Command
	if suggestion.Tool == "ansible" && strings.Contains(command, "playbook.yml") {
		if playbookFile := findPlaybookFile(); playbookFile != "" {
			command = strings.Replace(command, "playbook.yml", playbookFile, 1)
			fmt.Printf("📝 ops0: Found playbook: %s\n", playbookFile)
		}
	}
	
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("\n❌ ops0: Command failed with error: %v\n", err)
	} else {
		fmt.Printf("\n✅ ops0: Command completed successfully!\n")
	}
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
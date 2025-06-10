package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"regexp"
	"flag"
)

// Version information (set by GoReleaser or git)
var (
	version = "v0.1.0"  // Update this manually for each release
	commit  = getCommit()
	date    = getBuildDate()
)

// getVersion tries to get version from git tag, fallback to "dev"
func getVersion() string {
	if v := os.Getenv("VERSION"); v != "" {
		return v
	}
	
	// Try to get git tag
	if cmd := exec.Command("git", "describe", "--tags", "--exact-match", "HEAD"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			return strings.TrimSpace(string(output))
		}
	}
	
	// Try to get latest tag with commit info
	if cmd := exec.Command("git", "describe", "--tags", "--always", "--dirty"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			return strings.TrimSpace(string(output))
		}
	}
	
	return "dev"
}

// getCommit tries to get git commit hash
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

// getBuildDate returns current build time
func getBuildDate() string {
	if d := os.Getenv("BUILD_DATE"); d != "" {
		return d
	}
	
	return "unknown"
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
}

func main() {
	// Handle version flag
	var showVersion bool
	var message string
	
	flag.BoolVar(&showVersion, "version", false, "show version information")
	flag.StringVar(&message, "m", "", "natural language command message")
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
		fmt.Println("ops0 - Natural Language DevOps CLI")
		fmt.Printf("Version: %s\n\n", version)
		fmt.Println("Usage: ops0 -m \"your natural language command\"")
		fmt.Println("       ops0 -version")
		fmt.Println("\nExamples:")
		fmt.Println("  ops0 -m \"i want to plan my iac code\"")
		fmt.Println("  ops0 -m \"deploy my infrastructure\"")
		fmt.Println("  ops0 -m \"run my ansible playbook\"")
		os.Exit(1)
	}

	fmt.Printf("🤖 ops0: Analyzing your request: \"%s\"\n\n", message)

	// Parse the natural language input
	suggestion := parseIntent(message)
	
	if suggestion == nil {
		fmt.Println("❌ ops0: I couldn't understand your request. Try being more specific about what you want to do.")
		return
	}

	// Present the suggestion interactively
	handleInteraction(suggestion)
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
		}
	}
	
	if matched, _ := regexp.MatchString(`apply.*terraform|deploy.*infrastructure|apply.*iac`, input); matched {
		return &CommandSuggestion{
			Tool:        "terraform",
			Command:     "terraform apply",
			Description: "This will apply your Terraform configuration and make the actual infrastructure changes.",
			Intent:      "apply infrastructure changes",
		}
	}
	
	if matched, _ := regexp.MatchString(`destroy.*infrastructure|tear.*down|terraform.*destroy`, input); matched {
		return &CommandSuggestion{
			Tool:        "terraform",
			Command:     "terraform destroy",
			Description: "This will destroy all resources managed by your Terraform configuration.",
			Intent:      "destroy infrastructure",
		}
	}

	// Ansible patterns
	if matched, _ := regexp.MatchString(`run.*playbook|execute.*ansible|ansible.*playbook`, input); matched {
		return &CommandSuggestion{
			Tool:        "ansible",
			Command:     "ansible-playbook playbook.yml",
			Description: "This will run your Ansible playbook to configure your servers.",
			Intent:      "run ansible playbook",
		}
	}
	
	if matched, _ := regexp.MatchString(`check.*ansible|dry.*run.*ansible|ansible.*check`, input); matched {
		return &CommandSuggestion{
			Tool:        "ansible",
			Command:     "ansible-playbook playbook.yml --check",
			Description: "This will do a dry run of your Ansible playbook without making actual changes.",
			Intent:      "check ansible playbook",
		}
	}

	return nil
}

func handleInteraction(suggestion *CommandSuggestion) {
	// Check if the tool is installed
	tool := &Tool{
		Name:       suggestion.Tool,
		CheckCmd:   suggestion.Tool + " --version",
		InstallCmd: getInstallCommand(suggestion.Tool),
	}
	
	tool.IsInstalled = checkToolInstalled(tool)
	
	// Interactive conversation
	fmt.Printf("💡 ops0: I can help you %s via `%s`. %s\n", 
		suggestion.Intent, suggestion.Command, suggestion.Description)
	fmt.Print("Would you like me to do this? (y/n): ")
	
	if !getUserConfirmation() {
		fmt.Println("👋 ops0: No problem! Let me know if you need help with anything else.")
		return
	}

	// Handle tool installation if needed
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

	// Execute the command
	executeCommand(suggestion)
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
	
	// Handle special cases where we might need to find files
	command := suggestion.Command
	if suggestion.Tool == "ansible" && strings.Contains(command, "playbook.yml") {
		// Try to find actual playbook files
		if playbookFile := findPlaybookFile(); playbookFile != "" {
			command = strings.Replace(command, "playbook.yml", playbookFile, 1)
			fmt.Printf("📝 ops0: Found playbook: %s\n", playbookFile)
		}
	}
	
	// Execute the command
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
	// Common playbook filenames
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
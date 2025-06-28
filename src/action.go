package main

import (
	"encoding/json"
	"fmt"
	"bufio"
	"regexp"
	"os"
	"os/exec"
	"strings"
)

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

func showWelcomeMessage() {
	fmt.Println()
	fmt.Println("  ██████╗ ██████╗ ███████╗ ██████╗ ")
	fmt.Println("  ██╔══██╗██╔══██╗██╔════╝██╔═══██╗")
	fmt.Println("  ██║  ██║██████╔╝███████╗██║   ██║")
	fmt.Println("  ██║  ██║██╔═══╝ ╚════██║██║   ██║")
	fmt.Println("  ██████╔╝██║     ███████║╚██████╔╝")
	fmt.Println("  ╚═════╝ ╚═╝     ╚══════╝ ╚═════╝ ")
	fmt.Println()
	fmt.Println("🤖 ⚡ 👉 Natural Language DevOps Automation & Troubleshooting Tool")
	fmt.Println()
	fmt.Println("ops0 is an AI-powered natural language DevOps CLI native to Claude AI")
	fmt.Println("with ansible, terraform, kubernetes, aws, azure and docker operations")
	fmt.Println("in a single cli. An open-source alternative to complex DevOps workflows,")
	fmt.Println("manual operations, etc.")
	fmt.Println()
	fmt.Println("Type 'quit' or 'exit' to leave interactive mode")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

func runInteractiveSession() {
	showWelcomeMessage()
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
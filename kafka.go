package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)	

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

	if !isToolInstalled("brew") && runtime.GOOS == "darwin" {
		fmt.Println(red + "❌ Homebrew is not installed. Please install it to use this feature." + reset)
		os.Exit(1)
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
			os.Exit(1)
		}

		// "not_found"
		fmt.Println(red + "❌ Kafka command-line tools not found in your PATH." + reset)
		fmt.Print("Do you have Kafka installed in a custom location? (y/n): ")
		if getUserConfirmation() {
			fmt.Print("Please enter the full path to your Kafka 'bin' directory: ")
			reader := bufio.NewReader(os.Stdin)
			providedPath, _ := reader.ReadString('\n')
			providedPath = strings.TrimSpace(providedPath)

			// Validate the provided path
			pathWithSh := filepath.Join(providedPath, "kafka-topics.sh")
			pathWithoutSh := filepath.Join(providedPath, "kafka-topics")

			if _, statErr := os.Stat(pathWithSh); statErr == nil {
				cmdPath = pathWithSh
			} else if _, statErr := os.Stat(pathWithoutSh); statErr == nil {
				cmdPath = pathWithoutSh
			} else {
				fmt.Printf(red+"❌ Could not find Kafka tools in the provided path: %s"+reset+"\n", providedPath)
				cmdPath = "" // Ensure cmdPath is empty to fall through to installation
			}
		}

		// If path wasn't found or user didn't provide one, ask to install.
		if cmdPath == "" {
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
			os.Exit(1)
		}
	}

	// 2. Test connection to the cluster
	fmt.Printf("Connecting to Kafka cluster at %s...\n", brokers)
	args := []string{"--bootstrap-server", brokers, "--list"}
	if commandConfig != "" {
		args = append(args, "--command-config", commandConfig)
	}
	testCmd := exec.Command(cmdPath, args...)
	output, err := testCmd.CombinedOutput()

	if err != nil {
		fmt.Printf(red+"❌ Could not connect to Kafka cluster. Please check your broker addresses and configuration."+reset+"\n\n")
		fmt.Println(bold + "Error details from Kafka tools:" + reset)
		// Print the captured output which contains the detailed Java error
		fmt.Println(string(output))
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
				// Replace the command with the correct path using the pattern from cmdPath
				parts := strings.Fields(suggestion.Command)
				if len(parts) > 0 {
					baseCmd := parts[0]
					// Extract the command name without any path or suffix
					cmdName := strings.TrimSuffix(filepath.Base(baseCmd), ".sh")
					
					// Use the same pattern as our detected cmdPath
					// If cmdPath ends with .sh, use that pattern; otherwise use without .sh
					var replacementCmd string
					if strings.HasSuffix(cmdPath, ".sh") {
						// Use .sh pattern
						replacementCmd = strings.Replace(cmdPath, "kafka-topics.sh", cmdName+".sh", 1)
					} else {
						// Use without .sh pattern
						replacementCmd = strings.Replace(cmdPath, "kafka-topics", cmdName, 1)
					}
					
					suggestion.Command = strings.Replace(suggestion.Command, baseCmd, replacementCmd, 1)
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
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"sort"
	"strings"
	"time"
)

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
package main
	
import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

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

	// 2. For Kafka commands on Linux, also check with .sh suffix via PATH
	var cmdWithSh string
	if runtime.GOOS == "linux" && strings.HasPrefix(cmd, "kafka-") {
		cmdWithSh = cmd + ".sh"
		path, err := exec.LookPath(cmdWithSh)
		if err == nil {
			return path, nil
		}
	}

	// 3. If not in PATH, check common alternative locations.
	var commonPaths []string
	if runtime.GOOS == "darwin" {
		commonPaths = []string{
			"/opt/homebrew/bin", // Apple Silicon
			"/usr/local/bin",    // Intel Macs
		}
	} else if runtime.GOOS == "linux" {
		// General paths, avoiding user-specific ones.
		commonPaths = []string{
			"/opt/kafka/bin",
			"/usr/local/kafka/bin",
		}
	}

	for _, p := range commonPaths {
		// Check for command without suffix
		fullPath := filepath.Join(p, cmd)
		if _, err := os.Stat(fullPath); err == nil {
			if runtime.GOOS == "darwin" {
				return fullPath, fmt.Errorf("found_not_in_path")
			}
			return fullPath, nil
		}
		// For Linux, also check with .sh for kafka commands in these paths
		if runtime.GOOS == "linux" && cmdWithSh != "" {
			fullPathSh := filepath.Join(p, cmdWithSh)
			if _, err := os.Stat(fullPathSh); err == nil {
				return fullPathSh, nil
			}
		}
	}

	// 4. Really not found.
	return "", fmt.Errorf("not_found")
}
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
		handleInteractiveLogAnalysis(suggestion)
		return
	}

	// Normalize tool name for installation
	toolName := suggestion.Tool
	if toolName == "aws-cli" {
		toolName = "aws"
	}
	
	// Skip installation check for system_admin as it uses built-in commands
	if toolName != "system_admin" {
		if !isToolInstalled(toolName) {
			fmt.Printf("\n" + yellow + "⚠️  Tool '%s' not found. Would you like to install it? (y/n): " + reset, toolName)
			if getUserConfirmation() {
				tool := &Tool{
					Name:       toolName,
					CheckCmd:   toolName + " --version",
					InstallCmd: getInstallCommand(toolName),
				}
				installTool(tool)
			} else {
				fmt.Println("❌ Installation cancelled. Cannot proceed without the required tool.")
				return
			}
		}
	}

	// Show operation details and prompt for confirmation
	fmt.Printf("\n💡 Operation: %s\nCommand: %s\nDescription: %s\n", suggestion.Intent, suggestion.Command, suggestion.Description)
	
	if suggestion.HasDryRun && suggestion.DryRunCommand != "" {
		fmt.Print("Would you like to do a dry run first? (y/n): ")
		if getUserConfirmation() {
			executeDryRun(suggestion)
			fmt.Print("\nProceed with actual execution? (y/n): ")
			if !getUserConfirmation() {
				fmt.Println("❌ Operation cancelled.")
				return
			}
		}
	}
	
	fmt.Print("Would you like to execute this operation? (y/n): ")
	if getUserConfirmation() {
		executeCommand(suggestion)
	} else {
		fmt.Println("❌ Operation cancelled.")
	}
}

func handleInteractiveLogAnalysis(suggestion *CommandSuggestion) {
	fmt.Println("\n🔍 Interactive Log Analysis")
	fmt.Println("═══════════════════════════")
	
	// Handle kubectl-specific log analysis
	if suggestion.Tool == "kubectl" {
		handleKubernetesLogAnalysis(suggestion)
		return
	}
	
	// Fetch logs
	fmt.Println("📋 Fetching logs...")
	cmd := exec.Command("bash", "-c", suggestion.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ Error fetching logs: %v\n", err)
		return
	}
	
	logs := string(output)
	if len(logs) == 0 {
		fmt.Println("⚠️  No logs found or empty log output.")
		return
	}
	
	// Show log preview
	fmt.Println("\n📄 Log Preview (last 20 lines):")
	fmt.Println("─" + strings.Repeat("─", 50))
	lines := strings.Split(logs, "\n")
	start := len(lines) - 20
	if start < 0 {
		start = 0
	}
	for i := start; i < len(lines); i++ {
		if lines[i] != "" {
			fmt.Println(lines[i])
		}
	}
	fmt.Println("─" + strings.Repeat("─", 50))
	
	// Analyze logs
	fmt.Println("\n🧠 Analyzing logs for issues...")
	analysis := analyzeLogsIntelligently(logs)
	
	fmt.Println("\n📊 Analysis Results:")
	fmt.Println("═══════════════════")
	fmt.Println(analysis.Summary)
	
	if len(analysis.Issues) > 0 {
		fmt.Println("\n🚨 Issues Found:")
		for i, issue := range analysis.Issues {
			fmt.Printf("\n%d. %s\n", i+1, issue.Description)
			fmt.Printf("   Severity: %s\n", issue.Severity)
			fmt.Printf("   Pattern: %s\n", issue.Pattern)
			if issue.Suggestion != "" {
				fmt.Printf("   Suggestion: %s\n", issue.Suggestion)
			}
		}
		
		// Interactive fix suggestions
		fmt.Println("\n🔧 Interactive Fix Options:")
		fmt.Println("══════════════════════════")
		
		for i, issue := range analysis.Issues {
			if issue.FixCommand != "" {
				fmt.Printf("\n%d. Fix: %s\n", i+1, issue.Description)
				fmt.Printf("   Command: %s\n", issue.FixCommand)
				fmt.Print("   Execute this fix? (y/n): ")
				if getUserConfirmation() {
					fmt.Printf("\n🚀 Executing fix for issue %d...\n", i+1)
					executeFixCommand(issue.FixCommand, issue.Description)
				}
			}
		}
		
		// Additional recommendations
		if len(analysis.Recommendations) > 0 {
			fmt.Println("\n💡 Additional Recommendations:")
			for _, rec := range analysis.Recommendations {
				fmt.Printf("• %s\n", rec)
			}
		}
	} else {
		fmt.Println("✅ No critical issues found in the logs.")
	}
	
	// Ask if user wants to see more logs or perform additional analysis
	fmt.Print("\n🔍 Would you like to see more logs or perform additional analysis? (y/n): ")
	if getUserConfirmation() {
		handleAdditionalLogAnalysis(suggestion, logs)
	}
}

func handleKubernetesLogAnalysis(suggestion *CommandSuggestion) {
	fmt.Println("\n☸️  Kubernetes Log Analysis")
	fmt.Println("═══════════════════════════")
	
	// Handle different kubectl log intents
	switch suggestion.Intent {
	case "list_pods_for_logs":
		handleListPodsForLogs(suggestion)
		return
	case "analyze_logs_realtime":
		handleKubernetesRealtimeLogs(suggestion)
		return
	default:
		// Standard kubectl log analysis
		handleStandardKubernetesLogs(suggestion)
	}
}

func handleListPodsForLogs(suggestion *CommandSuggestion) {
	fmt.Println("📋 Listing pods to select for log analysis...")
	
	cmd := exec.Command("bash", "-c", suggestion.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ Error listing pods: %v\n", err)
		return
	}
	
	fmt.Println(string(output))
	
	fmt.Print("\n🔍 Enter pod name to analyze logs: ")
	reader := bufio.NewReader(os.Stdin)
	podName, _ := reader.ReadString('\n')
	podName = strings.TrimSpace(podName)
	
	if podName == "" {
		fmt.Println("❌ No pod name provided.")
		return
	}
	
	// Extract namespace from original command
	nsRe := regexp.MustCompile(`-n\s+([a-zA-Z0-9-]+)`)
	ns := "default"
	if m := nsRe.FindStringSubmatch(suggestion.Command); len(m) > 1 {
		ns = m[1]
	}
	
	// Create new suggestion for pod log analysis
	podSuggestion := &CommandSuggestion{
		Tool:        "kubectl",
		Command:     "kubectl logs " + podName + " -n " + ns + " --tail=100",
		Description: "Fetch and analyze the last 100 log lines for pod '" + podName + "' in namespace '" + ns + "'.",
		Intent:      "analyze_logs",
		Confidence:  0.95,
		AIGenerated: false,
		HasDryRun:   false,
	}
	
	handleStandardKubernetesLogs(podSuggestion)
}

func handleKubernetesRealtimeLogs(suggestion *CommandSuggestion) {
	fmt.Println("📺 Real-time Kubernetes Log Monitoring")
	fmt.Println("Press Ctrl+C to stop monitoring...")
	
	// Extract pod and namespace info for context
	podRe := regexp.MustCompile(`kubectl logs ([a-zA-Z0-9-]+)`)
	nsRe := regexp.MustCompile(`-n ([a-zA-Z0-9-]+)`)
	
	pod := ""
	ns := "default"
	if m := podRe.FindStringSubmatch(suggestion.Command); len(m) > 1 {
		pod = m[1]
	}
	if m := nsRe.FindStringSubmatch(suggestion.Command); len(m) > 1 {
		ns = m[1]
	}
	
	fmt.Printf("🔍 Monitoring logs for pod '%s' in namespace '%s'\n", pod, ns)
	
	// Show some context first
	fmt.Println("\n📋 Pod Status:")
	statusCmd := exec.Command("bash", "-c", "kubectl get pod "+pod+" -n "+ns+" -o wide")
	statusOutput, _ := statusCmd.CombinedOutput()
	fmt.Println(string(statusOutput))
	
	fmt.Println("\n📺 Starting real-time log monitoring...")
	
	cmd := exec.Command("bash", "-c", suggestion.Command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Start(); err != nil {
		fmt.Printf("❌ Error starting monitoring: %v\n", err)
		return
	}
	
	// Wait for user to interrupt
	cmd.Wait()
}

func handleStandardKubernetesLogs(suggestion *CommandSuggestion) {
	fmt.Println("📋 Fetching Kubernetes logs...")
	
	cmd := exec.Command("bash", "-c", suggestion.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ Error fetching logs: %v\n", err)
		return
	}
	
	logs := string(output)
	if len(logs) == 0 {
		fmt.Println("⚠️  No logs found or empty log output.")
		return
	}
	
	// Show log preview
	fmt.Println("\n📄 Log Preview (last 20 lines):")
	fmt.Println("─" + strings.Repeat("─", 50))
	lines := strings.Split(logs, "\n")
	start := len(lines) - 20
	if start < 0 {
		start = 0
	}
	for i := start; i < len(lines); i++ {
		if lines[i] != "" {
			fmt.Println(lines[i])
		}
	}
	fmt.Println("─" + strings.Repeat("─", 50))
	
	// Analyze logs with Kubernetes-specific patterns
	fmt.Println("\n🧠 Analyzing Kubernetes logs for issues...")
	analysis := analyzeKubernetesLogsIntelligently(logs)
	
	fmt.Println("\n📊 Analysis Results:")
	fmt.Println("═══════════════════")
	fmt.Println(analysis.Summary)
	
	if len(analysis.Issues) > 0 {
		fmt.Println("\n🚨 Issues Found:")
		for i, issue := range analysis.Issues {
			fmt.Printf("\n%d. %s\n", i+1, issue.Description)
			fmt.Printf("   Severity: %s\n", issue.Severity)
			fmt.Printf("   Pattern: %s\n", issue.Pattern)
			if issue.Suggestion != "" {
				fmt.Printf("   Suggestion: %s\n", issue.Suggestion)
			}
		}
		
		// Interactive fix suggestions for Kubernetes
		fmt.Println("\n🔧 Interactive Fix Options:")
		fmt.Println("══════════════════════════")
		
		for i, issue := range analysis.Issues {
			if issue.FixCommand != "" {
				fmt.Printf("\n%d. Fix: %s\n", i+1, issue.Description)
				fmt.Printf("   Command: %s\n", issue.FixCommand)
				fmt.Print("   Execute this fix? (y/n): ")
				if getUserConfirmation() {
					fmt.Printf("\n🚀 Executing fix for issue %d...\n", i+1)
					executeFixCommand(issue.FixCommand, issue.Description)
				}
			}
		}
		
		// Additional recommendations
		if len(analysis.Recommendations) > 0 {
			fmt.Println("\n💡 Additional Recommendations:")
			for _, rec := range analysis.Recommendations {
				fmt.Printf("• %s\n", rec)
			}
		}
	} else {
		fmt.Println("✅ No critical issues found in the Kubernetes logs.")
	}
	
	// Ask if user wants to see more logs or perform additional analysis
	fmt.Print("\n🔍 Would you like to see more logs or perform additional analysis? (y/n): ")
	if getUserConfirmation() {
		handleAdditionalKubernetesAnalysis(suggestion, logs)
	}
}

func analyzeKubernetesLogsIntelligently(logs string) LogAnalysis {
	analysis := LogAnalysis{
		Summary: "Kubernetes log analysis completed.",
		Issues:  []LogIssue{},
		Recommendations: []string{},
	}
	
	lines := strings.Split(logs, "\n")
	
	// Analyze for Kubernetes-specific issues
	issues := detectKubernetesIssues(lines)
	analysis.Issues = append(analysis.Issues, issues...)
	
	// Generate summary
	if len(issues) == 0 {
		analysis.Summary = "✅ Kubernetes logs appear healthy with no critical issues detected."
	} else {
		analysis.Summary = fmt.Sprintf("⚠️  Found %d potential Kubernetes issues that may require attention.", len(issues))
	}
	
	// Add Kubernetes-specific recommendations
	analysis.Recommendations = generateKubernetesRecommendations(lines, issues)
	
	return analysis
}

func detectKubernetesIssues(lines []string) []LogIssue {
	var issues []LogIssue
	
	// Kubernetes-specific error patterns
	k8sErrorPatterns := map[string]string{
		"connection refused": "Kubernetes service connectivity issue",
		"permission denied":  "Kubernetes RBAC/permission issue",
		"not found":          "Kubernetes resource not found",
		"timeout":            "Kubernetes operation timeout",
		"out of memory":      "Kubernetes pod memory exhaustion",
		"disk full":          "Kubernetes pod storage issue",
		"crashloopbackoff":   "Pod crash loop detected",
		"imagepullbackoff":   "Container image pull failure",
		"pending":            "Pod stuck in pending state",
		"failed":             "Pod failed to start",
		"oomkilled":          "Pod killed due to OOM",
		"liveness probe failed": "Liveness probe failure",
		"readiness probe failed": "Readiness probe failure",
		"tls handshake":      "TLS/SSL certificate issue",
		"certificate":        "Certificate validation issue",
	}
	
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		
		for pattern, description := range k8sErrorPatterns {
			if strings.Contains(lineLower, pattern) {
				severity := "MEDIUM"
				if strings.Contains(pattern, "crashloopbackoff") || strings.Contains(pattern, "oomkilled") || 
				   strings.Contains(pattern, "imagepullbackoff") {
					severity = "HIGH"
				}
				
				fixCommand := generateKubernetesFixCommand(pattern, line)
				
				issues = append(issues, LogIssue{
					Description: description,
					Severity:    severity,
					Pattern:     pattern,
					Suggestion:  getKubernetesSuggestionForPattern(pattern),
					FixCommand:  fixCommand,
				})
			}
		}
	}
	
	return issues
}

func generateKubernetesFixCommand(pattern, context string) string {
	// Extract pod and namespace from context if available
	podRe := regexp.MustCompile(`pod[:\s]+([a-zA-Z0-9-]+)`)
	nsRe := regexp.MustCompile(`namespace[:\s]+([a-zA-Z0-9-]+)`)
	
	pod := ""
	ns := "default"
	if m := podRe.FindStringSubmatch(context); len(m) > 1 {
		pod = m[1]
	}
	if m := nsRe.FindStringSubmatch(context); len(m) > 1 {
		ns = m[1]
	}
	
	switch {
	case strings.Contains(pattern, "crashloopbackoff"):
		if pod != "" {
			return fmt.Sprintf("kubectl describe pod %s -n %s", pod, ns)
		}
		return "kubectl get pods --all-namespaces | grep CrashLoopBackOff"
	case strings.Contains(pattern, "imagepullbackoff"):
		if pod != "" {
			return fmt.Sprintf("kubectl describe pod %s -n %s", pod, ns)
		}
		return "kubectl get pods --all-namespaces | grep ImagePullBackOff"
	case strings.Contains(pattern, "pending"):
		if pod != "" {
			return fmt.Sprintf("kubectl describe pod %s -n %s", pod, ns)
		}
		return "kubectl get pods --all-namespaces | grep Pending"
	case strings.Contains(pattern, "oomkilled"):
		if pod != "" {
			return fmt.Sprintf("kubectl top pod %s -n %s", pod, ns)
		}
		return "kubectl top pods --all-namespaces"
	case strings.Contains(pattern, "liveness probe failed") || strings.Contains(pattern, "readiness probe failed"):
		if pod != "" {
			return fmt.Sprintf("kubectl describe pod %s -n %s", pod, ns)
		}
		return "kubectl get events --all-namespaces --sort-by='.lastTimestamp'"
	case strings.Contains(pattern, "connection refused"):
		return "kubectl get svc --all-namespaces"
	case strings.Contains(pattern, "permission denied"):
		return "kubectl auth can-i --list"
	case strings.Contains(pattern, "tls handshake") || strings.Contains(pattern, "certificate"):
		return "kubectl get secrets --all-namespaces | grep tls"
	default:
		if pod != "" {
			return fmt.Sprintf("kubectl describe pod %s -n %s", pod, ns)
		}
		return "kubectl get events --all-namespaces --sort-by='.lastTimestamp'"
	}
}

func getKubernetesSuggestionForPattern(pattern string) string {
	suggestions := map[string]string{
		"crashloopbackoff":   "Check pod events and container logs for the root cause of crashes",
		"imagepullbackoff":   "Verify image name, registry credentials, and network connectivity",
		"pending":            "Check resource availability, node selectors, and taints/tolerations",
		"oomkilled":          "Increase memory limits or optimize application memory usage",
		"liveness probe failed": "Check if the application is responding on the probe port",
		"readiness probe failed": "Verify the application is ready to serve traffic",
		"connection refused": "Check if the target service exists and is running",
		"permission denied":  "Verify RBAC policies and service account permissions",
		"tls handshake":      "Check TLS certificate configuration and validity",
		"certificate":        "Verify certificate configuration and expiration",
	}
	
	if suggestion, exists := suggestions[pattern]; exists {
		return suggestion
	}
	return "Investigate the root cause using kubectl describe and kubectl logs"
}

func generateKubernetesRecommendations(lines []string, issues []LogIssue) []string {
	var recommendations []string
	
	// Count error types
	errorCount := len(issues)
	if errorCount > 10 {
		recommendations = append(recommendations, "High error volume detected - consider implementing better monitoring and alerting")
	}
	
	// Check for repeated patterns
	patterns := make(map[string]int)
	for _, issue := range issues {
		patterns[issue.Pattern]++
	}
	
	for pattern, count := range patterns {
		if count > 5 {
			recommendations = append(recommendations, fmt.Sprintf("Frequent '%s' errors - investigate root cause", pattern))
		}
	}
	
	// Kubernetes-specific recommendations
	if len(issues) > 0 {
		recommendations = append(recommendations, "Set up Kubernetes monitoring with Prometheus and Grafana")
		recommendations = append(recommendations, "Implement proper resource limits and requests")
		recommendations = append(recommendations, "Consider using health checks and proper probe configurations")
	}
	
	return recommendations
}

func handleAdditionalKubernetesAnalysis(suggestion *CommandSuggestion, currentLogs string) {
	fmt.Println("\n🔍 Additional Kubernetes Analysis Options:")
	fmt.Println("1. Show more log lines")
	fmt.Println("2. Search for specific patterns")
	fmt.Println("3. Analyze error frequency")
	fmt.Println("4. Check pod status and events")
	fmt.Println("5. Monitor logs in real-time")
	fmt.Println("6. Check resource usage")
	fmt.Print("\nSelect option (1-6): ")
	
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	
	switch choice {
	case "1":
		showMoreLogs(suggestion)
	case "2":
		searchLogPatterns(currentLogs)
	case "3":
		analyzeErrorFrequency(currentLogs)
	case "4":
		checkKubernetesPodStatus(suggestion)
	case "5":
		monitorLogsRealTime(suggestion)
	case "6":
		checkKubernetesResourceUsage(suggestion)
	default:
		fmt.Println("Invalid option selected.")
	}
}

func checkKubernetesPodStatus(suggestion *CommandSuggestion) {
	// Extract pod and namespace from command
	podRe := regexp.MustCompile(`kubectl logs ([a-zA-Z0-9-]+)`)
	nsRe := regexp.MustCompile(`-n ([a-zA-Z0-9-]+)`)
	
	pod := ""
	ns := "default"
	if m := podRe.FindStringSubmatch(suggestion.Command); len(m) > 1 {
		pod = m[1]
	}
	if m := nsRe.FindStringSubmatch(suggestion.Command); len(m) > 1 {
		ns = m[1]
	}
	
	fmt.Println("\n📊 Pod Status and Events:")
	fmt.Println("═══════════════════════════")
	
	if pod != "" {
		// Show pod details
		describeCmd := exec.Command("bash", "-c", fmt.Sprintf("kubectl describe pod %s -n %s", pod, ns))
		describeOutput, _ := describeCmd.CombinedOutput()
		fmt.Println(string(describeOutput))
	} else {
		// Show all pods in namespace
		podsCmd := exec.Command("bash", "-c", fmt.Sprintf("kubectl get pods -n %s -o wide", ns))
		podsOutput, _ := podsCmd.CombinedOutput()
		fmt.Println(string(podsOutput))
	}
	
	// Show recent events
	fmt.Println("\n📅 Recent Events:")
	eventsCmd := exec.Command("bash", "-c", fmt.Sprintf("kubectl get events -n %s --sort-by='.lastTimestamp' | tail -20", ns))
	eventsOutput, _ := eventsCmd.CombinedOutput()
	fmt.Println(string(eventsOutput))
}

func checkKubernetesResourceUsage(suggestion *CommandSuggestion) {
	// Extract namespace from command
	nsRe := regexp.MustCompile(`-n ([a-zA-Z0-9-]+)`)
	ns := "default"
	if m := nsRe.FindStringSubmatch(suggestion.Command); len(m) > 1 {
		ns = m[1]
	}
	
	fmt.Println("\n📊 Resource Usage:")
	fmt.Println("═══════════════════")
	
	// Check if metrics-server is available
	topCmd := exec.Command("bash", "-c", fmt.Sprintf("kubectl top pods -n %s", ns))
	topOutput, err := topCmd.CombinedOutput()
	if err != nil {
		fmt.Println("⚠️  Metrics server not available. Install metrics-server to see resource usage.")
		fmt.Println("   Run: kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml")
	} else {
		fmt.Println(string(topOutput))
	}
	
	// Show resource requests and limits
	fmt.Println("\n📋 Resource Requests and Limits:")
	resourceCmd := exec.Command("bash", "-c", fmt.Sprintf("kubectl get pods -n %s -o custom-columns=NAME:.metadata.name,CPU_REQ:.spec.containers[0].resources.requests.cpu,CPU_LIMIT:.spec.containers[0].resources.limits.cpu,MEM_REQ:.spec.containers[0].resources.requests.memory,MEM_LIMIT:.spec.containers[0].resources.limits.memory", ns))
	resourceOutput, _ := resourceCmd.CombinedOutput()
	fmt.Println(string(resourceOutput))
}

func handleAdditionalLogAnalysis(suggestion *CommandSuggestion, currentLogs string) {
	fmt.Println("\n🔍 Additional Analysis Options:")
	fmt.Println("1. Show more log lines")
	fmt.Println("2. Search for specific patterns")
	fmt.Println("3. Analyze error frequency")
	fmt.Println("4. Check for performance issues")
	fmt.Println("5. Monitor logs in real-time")
	fmt.Print("\nSelect option (1-5): ")
	
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	
	switch choice {
	case "1":
		showMoreLogs(suggestion)
	case "2":
		searchLogPatterns(currentLogs)
	case "3":
		analyzeErrorFrequency(currentLogs)
	case "4":
		analyzePerformanceIssues(currentLogs)
	case "5":
		monitorLogsRealTime(suggestion)
	default:
		fmt.Println("Invalid option selected.")
	}
}

func showMoreLogs(suggestion *CommandSuggestion) {
	fmt.Print("How many lines to show? (default 100): ")
	reader := bufio.NewReader(os.Stdin)
	linesStr, _ := reader.ReadString('\n')
	linesStr = strings.TrimSpace(linesStr)
	
	lines := "100"
	if linesStr != "" {
		lines = linesStr
	}
	
	// Modify command to show more lines
	modifiedCmd := strings.Replace(suggestion.Command, "--tail=100", "--tail="+lines, 1)
	modifiedCmd = strings.Replace(modifiedCmd, "-n 100", "-n "+lines, 1)
	
	cmd := exec.Command("bash", "-c", modifiedCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	
	fmt.Printf("\n📄 Showing last %s lines:\n", lines)
	fmt.Println("─" + strings.Repeat("─", 50))
	fmt.Println(string(output))
	fmt.Println("─" + strings.Repeat("─", 50))
}

func searchLogPatterns(logs string) {
	fmt.Print("Enter search pattern (e.g., 'error', 'timeout', 'connection'): ")
	reader := bufio.NewReader(os.Stdin)
	pattern, _ := reader.ReadString('\n')
	pattern = strings.TrimSpace(pattern)
	
	if pattern == "" {
		fmt.Println("No pattern provided.")
		return
	}
	
	fmt.Printf("\n🔍 Searching for pattern: '%s'\n", pattern)
	fmt.Println("─" + strings.Repeat("─", 50))
	
	lines := strings.Split(logs, "\n")
	count := 0
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
			fmt.Println(line)
			count++
		}
	}
	
	if count == 0 {
		fmt.Println("No matches found.")
	} else {
		fmt.Printf("\nFound %d matches.\n", count)
	}
	fmt.Println("─" + strings.Repeat("─", 50))
}

func analyzeErrorFrequency(logs string) {
	fmt.Println("\n📊 Error Frequency Analysis:")
	fmt.Println("═══════════════════════════")
	
	errorPatterns := map[string]int{
		"error":     0,
		"exception": 0,
		"failed":    0,
		"timeout":   0,
		"connection refused": 0,
		"permission denied": 0,
		"not found": 0,
	}
	
	lines := strings.Split(logs, "\n")
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		for pattern := range errorPatterns {
			if strings.Contains(lineLower, pattern) {
				errorPatterns[pattern]++
			}
		}
	}
	
	totalErrors := 0
	for pattern, count := range errorPatterns {
		if count > 0 {
			fmt.Printf("• %s: %d occurrences\n", pattern, count)
			totalErrors += count
		}
	}
	
	if totalErrors == 0 {
		fmt.Println("✅ No errors found in the logs.")
	} else {
		fmt.Printf("\nTotal error occurrences: %d\n", totalErrors)
	}
}

func analyzePerformanceIssues(logs string) {
	fmt.Println("\n⚡ Performance Analysis:")
	fmt.Println("═══════════════════════")
	
	performancePatterns := []string{
		"slow", "timeout", "latency", "response time", "memory", "cpu", "load",
		"connection pool", "deadlock", "lock", "wait", "blocked",
	}
	
	lines := strings.Split(logs, "\n")
	performanceIssues := []string{}
	
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		for _, pattern := range performancePatterns {
			if strings.Contains(lineLower, pattern) {
				performanceIssues = append(performanceIssues, line)
				break
			}
		}
	}
	
	if len(performanceIssues) == 0 {
		fmt.Println("✅ No obvious performance issues detected.")
	} else {
		fmt.Printf("⚠️  Found %d potential performance-related entries:\n", len(performanceIssues))
		for i, issue := range performanceIssues {
			if i < 10 { // Limit to first 10
				fmt.Printf("• %s\n", issue)
			}
		}
		if len(performanceIssues) > 10 {
			fmt.Printf("... and %d more\n", len(performanceIssues)-10)
		}
	}
}

func monitorLogsRealTime(suggestion *CommandSuggestion) {
	fmt.Println("\n📺 Real-time Log Monitoring")
	fmt.Println("Press Ctrl+C to stop monitoring...")
	
	// Modify command for real-time monitoring
	monitorCmd := strings.Replace(suggestion.Command, "--tail=100", "-f", 1)
	monitorCmd = strings.Replace(monitorCmd, "-n 100", "-f", 1)
	
	cmd := exec.Command("bash", "-c", monitorCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Start(); err != nil {
		fmt.Printf("❌ Error starting monitoring: %v\n", err)
		return
	}
	
	// Wait for user to interrupt
	cmd.Wait()
}

func executeFixCommand(command, description string) {
	fmt.Printf("Executing: %s\n", command)
	
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Fix command failed: %v\n", err)
	} else {
		fmt.Printf("✅ Fix applied successfully!\n")
	}
}

type LogIssue struct {
	Description  string
	Severity     string
	Pattern      string
	Suggestion   string
	FixCommand   string
}

type LogAnalysis struct {
	Summary        string
	Issues         []LogIssue
	Recommendations []string
}

func analyzeLogsIntelligently(logs string) LogAnalysis {
	analysis := LogAnalysis{
		Summary: "Log analysis completed.",
		Issues:  []LogIssue{},
		Recommendations: []string{},
	}
	
	lines := strings.Split(logs, "\n")
	
	// Analyze for common issues
	issues := detectCommonIssues(lines)
	analysis.Issues = append(analysis.Issues, issues...)
	
	// Generate summary
	if len(issues) == 0 {
		analysis.Summary = "✅ Logs appear healthy with no critical issues detected."
	} else {
		analysis.Summary = fmt.Sprintf("⚠️  Found %d potential issues that may require attention.", len(issues))
	}
	
	// Add recommendations based on context
	analysis.Recommendations = generateRecommendations(lines, issues)
	
	return analysis
}

func detectCommonIssues(lines []string) []LogIssue {
	var issues []LogIssue
	
	// Error patterns
	errorPatterns := map[string]string{
		"connection refused": "Network connectivity issue",
		"permission denied":  "Permission/access issue",
		"not found":          "Resource not found",
		"timeout":            "Operation timeout",
		"out of memory":      "Memory exhaustion",
		"disk full":          "Storage space issue",
		"deadlock":           "Database deadlock",
		"ssl certificate":    "SSL/TLS certificate issue",
	}
	
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		
		for pattern, description := range errorPatterns {
			if strings.Contains(lineLower, pattern) {
				severity := "MEDIUM"
				if strings.Contains(pattern, "out of memory") || strings.Contains(pattern, "disk full") {
					severity = "HIGH"
				}
				
				fixCommand := generateFixCommand(pattern, line)
				
				issues = append(issues, LogIssue{
					Description: description,
					Severity:    severity,
					Pattern:     pattern,
					Suggestion:  getSuggestionForPattern(pattern),
					FixCommand:  fixCommand,
				})
			}
		}
	}
	
	return issues
}

func generateFixCommand(pattern, context string) string {
	switch {
	case strings.Contains(pattern, "connection refused"):
		return "netstat -tuln | grep LISTEN"
	case strings.Contains(pattern, "permission denied"):
		return "ls -la " + extractPathFromContext(context)
	case strings.Contains(pattern, "not found"):
		return "find / -name " + extractResourceFromContext(context) + " 2>/dev/null"
	case strings.Contains(pattern, "timeout"):
		return "ping -c 3 " + extractHostFromContext(context)
	case strings.Contains(pattern, "out of memory"):
		return "free -h && ps aux --sort=-%mem | head -10"
	case strings.Contains(pattern, "disk full"):
		return "df -h"
	case strings.Contains(pattern, "ssl certificate"):
		return "openssl s_client -connect " + extractHostFromContext(context) + ":443 -servername " + extractHostFromContext(context)
	default:
		return ""
	}
}

func getSuggestionForPattern(pattern string) string {
	suggestions := map[string]string{
		"connection refused": "Check if the service is running and listening on the expected port",
		"permission denied":  "Verify file permissions and user access rights",
		"not found":          "Ensure the resource exists and check the path",
		"timeout":            "Check network connectivity and service responsiveness",
		"out of memory":      "Monitor memory usage and consider increasing available memory",
		"disk full":          "Free up disk space or expand storage",
		"ssl certificate":    "Verify SSL certificate validity and expiration",
	}
	
	if suggestion, exists := suggestions[pattern]; exists {
		return suggestion
	}
	return "Investigate the root cause of this issue"
}

func generateRecommendations(lines []string, issues []LogIssue) []string {
	var recommendations []string
	
	// Count error types
	errorCount := len(issues)
	if errorCount > 10 {
		recommendations = append(recommendations, "High error volume detected - consider implementing better error handling and monitoring")
	}
	
	// Check for repeated patterns
	patterns := make(map[string]int)
	for _, issue := range issues {
		patterns[issue.Pattern]++
	}
	
	for pattern, count := range patterns {
		if count > 5 {
			recommendations = append(recommendations, fmt.Sprintf("Frequent '%s' errors - investigate root cause", pattern))
		}
	}
	
	// General recommendations
	if len(issues) > 0 {
		recommendations = append(recommendations, "Set up log monitoring and alerting for critical errors")
		recommendations = append(recommendations, "Consider implementing automated recovery procedures")
	}
	
	return recommendations
}

func extractPathFromContext(context string) string {
	// Simple extraction - look for paths in the context
	re := regexp.MustCompile(`(/[^\s]+)`)
	if match := re.FindString(context); match != "" {
		return match
	}
	return "/tmp"
}

func extractResourceFromContext(context string) string {
	// Extract resource name from context
	re := regexp.MustCompile(`([a-zA-Z0-9_-]+\.(conf|yml|yaml|json|log))`)
	if match := re.FindString(context); match != "" {
		return match
	}
	return "resource"
}

func extractHostFromContext(context string) string {
	// Extract host/IP from context
	re := regexp.MustCompile(`(\b(?:\d{1,3}\.){3}\d{1,3}\b|[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
	if match := re.FindString(context); match != "" {
		return match
	}
	return "localhost"
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
				// Use the enhanced interactive log analysis
				handleInteractiveLogAnalysis(suggestion)
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
package main

import (
	"strings"
	"regexp"
)

func ParseIntent(input string) *CommandSuggestion {
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
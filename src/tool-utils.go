package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

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
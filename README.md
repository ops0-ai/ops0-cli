<p align="center">
  <img src="assets/logo.jpg" alt="ops0 CLI Logo" width="150">
</p>

<p align="center">
  <a href="./README.zh-CN.md">ReadMe in Chinese</a> • 
  <a href="./README.de.md">ReadMe in German</a> • 
  <a href="./README.fr.md">ReadMe in French</a> • 
  <a href="./README.es.md">ReadMe in Spanish</a> • 
  <a href="./README.pt-BR.md">ReadMe in Portuguese</a> • 
  <a href="https://join.slack.com/t/ops0/shared_invite/zt-37akwqb1v-BvfK7AioDlRhje94UN2tkw">Slack Community</a>
</p>

---

<p align="center">
ops0 is an intelligent CLI tool that transforms natural language into DevOps operations.<br>
Powered by Claude AI, it simplifies complex DevOps tasks by understanding your intent<br>
and performing the right operations, making DevOps management more accessible and efficient.
</p>

## ops0 in Action

![ops0 CLI Demo](assets/ops0cli.gif)
*Watch ops0 translate natural language into powerful DevOps operations*

## Quick Start

### Installation
```bash
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | bash
```

### Basic Usage
```bash
# Rule-based mode (works without API key)
ops0 -m "i want to plan my iac code"

# AI-powered mode (requires API key)
export ANTHROPIC_API_KEY=your_key_here
ops0 -m "check if my kubernetes pods are running" -ai

# Troubleshooting mode
ops0 -m "my terraform apply is failing with state lock" -troubleshoot

# Interactive operations mode
ops0 -o
```

*Use `ops0 -o` for interactive operations mode: enter natural language requests and run multiple operations in a session until you type 'quit' or 'exit'.*

## Operations in Action

Here are some real-world examples of ops0 in action across different tools:

### Kafka Admin Mode
[Watch a video of ops0's Kafka Admin Mode in action](https://www.loom.com/share/c800f1f15865489780586c9d154ef365?sid=0e17ade7-6035-4eea-853a-c0e924ec4715)

*Example: Interactively manage Kafka clusters using natural language.*

### ops0 statistics
<img width="1022" alt="image" src="https://github.com/user-attachments/assets/000b01a3-6565-432d-96f1-cd64d912475c" />

### AWS CLI Operations
![AWS CLI Example](assets/aws.png)
*Example: Managing AWS resources using natural language commands*

### Docker Container Management
![Docker Example](assets/docker.png)
*Example: Managing Docker containers and images with simple English*

### Ansible Automation
![Ansible Example](assets/ansible.png)
![Ansible Playbook](assets/ansible-playbook.png)
*Example: Executing and validating Ansible playbooks effortlessly*

### Terraform Infrastructure
![Terraform Example](assets/terraform.png)
*Example: Managing infrastructure as code with natural language*

### Kubernetes Operations
![Kubernetes Example](assets/kubernetes.png)
*Example: Simplified Kubernetes cluster management and troubleshooting*

### Install All Tools at Once

![CLI Install All Tools](assets/cli-install.png)

You can now install all supported DevOps tools with a single command:

```bash
ops0 --install
```

This will automatically install Terraform, Ansible, kubectl, Docker, Helm, AWS CLI, gcloud, and Azure CLI, and show their versions in a summary table.

Each example demonstrates:
- Natural language command input
- AI-powered command translation
- Clear command preview
- Safe execution with confirmation
- Detailed output formatting

## Supported Tools & Features

### Core Tools
- **Terraform** - Infrastructure as Code
- **Ansible** - Configuration Management
- **Kubernetes (kubectl)** - Container Orchestration
- **Docker** - Containerization
- **AWS CLI** - Amazon Web Services
- **Helm** - Kubernetes Package Manager
- **gcloud** - Google Cloud Platform
- **Azure CLI** - Microsoft Azure
- **System Admin** - Linux System Administration

### System Administration Examples
```bash
# Monitor system resources
ops0 -m "show memory usage on my machine"
ops0 -m "check disk space"
ops0 -m "display CPU usage"

# Manage system services
ops0 -m "restart service nginx"
ops0 -m "check service status apache2"

# Package management
ops0 -m "install package docker"
ops0 -m "update system packages"

# System logs
ops0 -m "show system logs"
ops0 -m "check journal logs"
```

### System Administration & Log Analysis Examples
```bash
# Analyze Kubernetes pod logs and get AI-powered summary and recommendations
ops0 -m "analyze logs for pod my-app-123 in namespace prod"

# Analyze a specific log file for issues
ops0 -m "analyze /var/log/nginx/error.log"
```

### Key Features
- Natural language command translation
- AI-powered troubleshooting
- **Kubernetes pod log analysis with AI summary and suggested operations**
- **Log file path analysis for issues and context**
- Context-aware suggestions
- Safe execution with confirmations
- Dry run support for destructive operations
- Automatic tool installation

## 🆚 AI vs Rule-Based Mode

| Feature | Rule-Based | AI Mode |
|---------|------------|---------|
| Setup | No API key needed | Requires ANTHROPIC_API_KEY |
| Speed | Instant | ~2-3 seconds |
| Understanding | Pattern matching | Natural language |
| Context Awareness | Limited | High |
| Troubleshooting | Basic | Advanced |
| Complex Scenarios | Limited | Excellent |
| Offline Usage | ✅ | ❌ |

## Configuration

### Environment Variables
```bash
# Required for AI features
export ANTHROPIC_API_KEY=your_api_key

# Optional: Customize AI behavior
export OPS0_AI_MODEL=claude-3-sonnet-20240229  # Default model
export OPS0_MAX_TOKENS=1024                    # Response length
```

### Config File (Future)
```yaml
# ~/.ops0/config.yaml
ai:
  provider: anthropic
  model: claude-3-sonnet-20240229
  max_tokens: 1024
  
tools:
  terraform:
    version_check: terraform version
    install_cmd: brew install terraform
  kubectl:
    version_check: kubectl version --client
    install_cmd: brew install kubectl
```

## Privacy & Security

- **API Key**: Stored locally as environment variable
- **No Data Storage**: Commands and context not stored by ops0
- **Anthropic Privacy**: Follows Anthropic's data handling policies
- **Local Processing**: Rule-based mode works completely offline

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Areas needing help
- New tool integrations
- AI prompt improvements
- Testing across different environments
- Documentation and examples

## 🗺️ Roadmap

### Current
- [x] Claude AI integration
- [x] Basic troubleshooting mode
- [x] Context awareness
- [x] Multi-tool support

### Coming Soon
- [ ] Custom offline-capable model for air-gapped environments
- [ ] Interactive multi-step workflows
- [ ] Learning from user feedback
- [ ] Custom tool configurations
- [ ] Multiple AI provider support
- [ ] Advanced context analysis
- [ ] Team collaboration features

## Tips

1. **Be Specific**: "my terraform plan shows 5 resources changing" vs "terraform error"
2. **Use Troubleshoot Mode**: For complex issues, use `-troubleshoot` flag
3. **Check Context**: AI works better when you're in the right directory
4. **Review Commands**: Always review AI suggestions before confirming
5. **Provide Feedback**: Use GitHub issues to report AI accuracy problems

# ops0 - AI-Powered Natural Language DevOps CLI

A smart CLI tool that translates natural language into DevOps commands, now powered by Claude AI for intelligent command generation and troubleshooting.

## 🚀 Quick Start

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
```

## 🧠 AI Features

### Setup
1. **Get Claude API Key**: Sign up at [console.anthropic.com](https://console.anthropic.com)
2. **Set Environment Variable**:
   ```bash
   export ANTHROPIC_API_KEY=your_api_key_here
   # Add to ~/.bashrc or ~/.zshrc for persistence
   echo 'export ANTHROPIC_API_KEY=your_api_key_here' >> ~/.bashrc
   ```

### AI Modes

#### **Standard AI Mode**
```bash
ops0 -m "deploy my application to kubernetes" -ai
```
- Better natural language understanding
- Context-aware command suggestions
- Support for complex scenarios
- Confidence scoring

#### **Troubleshooting Mode**
```bash
ops0 -m "my pods keep crashing" -troubleshoot
```
- Analyzes system context
- Suggests diagnostic commands
- Step-by-step troubleshooting
- Gathers environment information

## 📖 Usage Examples

### Infrastructure Management
```bash
# Basic Terraform
ops0 -m "plan my infrastructure changes"
ops0 -m "apply my terraform configuration"
ops0 -m "destroy my test environment"

# AI-Enhanced
ops0 -m "I want to see what will change before deploying my AWS infrastructure" -ai
ops0 -m "safely deploy my infrastructure with approval" -ai
```

### Kubernetes Operations
```bash
# AI understands context better
ops0 -m "show me all my pods and their status" -ai
ops0 -m "get logs from the failing pod" -ai
ops0 -m "scale my deployment to 3 replicas" -ai
```

### Troubleshooting
```bash
# Diagnostic mode
ops0 -m "my application is not responding" -troubleshoot
ops0 -m "pods are in pending state" -troubleshoot
ops0 -m "terraform state is locked" -troubleshoot
```

### Docker & Containers
```bash
ops0 -m "build my docker image" -ai
ops0 -m "check running containers and their resource usage" -ai
ops0 -m "my container keeps restarting" -troubleshoot
```

## 🛠️ Supported Tools

### Core Tools (Rule-based + AI)
- **Terraform** - Infrastructure as Code
- **Ansible** - Configuration Management
- **Kubernetes (kubectl)** - Container Orchestration
- **Docker** - Containerization
- **Helm** - Kubernetes Package Manager

### AI-Enhanced Tools
- **AWS CLI** - Amazon Web Services
- **gcloud** - Google Cloud Platform
- **Azure CLI** - Microsoft Azure
- **Advanced troubleshooting** for all tools

## 🔧 Configuration

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

## 🔍 How AI Mode Works

1. **Context Gathering**: Scans your environment for installed tools, project files, and current directory
2. **Intent Analysis**: Uses Claude to understand your natural language request
3. **Command Generation**: Creates appropriate commands with explanations
4. **Safety Checks**: Prefers safe, diagnostic commands when possible
5. **Interactive Confirmation**: Always asks before executing commands

## 🎯 AI Capabilities

### Natural Language Understanding
```bash
# Instead of remembering exact syntax:
ops0 -m "show me what's wrong with my kubernetes deployment" -ai

# AI translates to:
# kubectl describe deployment <deployment-name>
# kubectl get pods -l app=<app-name>
# kubectl logs deployment/<deployment-name>
```

### Context-Aware Suggestions
```bash
# AI considers your current directory and available tools
ops0 -m "deploy this" -ai

# In a directory with Dockerfile -> docker build/run
# In a directory with terraform files -> terraform plan/apply
# In a directory with k8s manifests -> kubectl apply
```

### Progressive Troubleshooting
```bash
ops0 -m "my app is down" -troubleshoot

# AI suggests step-by-step:
# 1. Check if pods are running
# 2. Get pod logs
# 3. Check service endpoints
# 4. Verify ingress configuration
```

## 🚨 Safety Features

- **Read-First Approach**: AI prefers diagnostic commands before making changes
- **Confirmation Required**: All commands require user approval
- **Confidence Scoring**: Shows how confident the AI is about suggestions
- **Fallback Mode**: Falls back to rule-based parsing if AI fails
- **No Destructive Defaults**: Never suggests dangerous operations without clear intent

## 🔧 Development

### Building with AI Support
```bash
# Clone and build
git clone https://github.com/ops0-ai/ops0-cli
cd ops0-cli
go build -o ops0 main.go

# Test AI mode
export ANTHROPIC_API_KEY=your_key
./ops0 -m "test ai functionality" -ai
```

### Adding New Tools
```go
// Add to parseIntent for rule-based support
if matched, _ := regexp.MatchString(`your_pattern`, input); matched {
    return &CommandSuggestion{
        Tool: "your_tool",
        Command: "your_command",
        // ...
    }
}
```

## 📊 Usage Analytics

ops0 learns from usage patterns to improve suggestions:
- Most common command patterns
- Success/failure rates
- User feedback (future feature)
- Context-specific optimizations

## 🛡️ Privacy & Security

- **API Key**: Stored locally as environment variable
- **No Data Storage**: Commands and context not stored by ops0
- **Anthropic Privacy**: Follows Anthropic's data handling policies
- **Local Processing**: Rule-based mode works completely offline

## 🗺️ Roadmap

### Phase 1 (Current)
- [x] Claude AI integration
- [x] Basic troubleshooting mode
- [x] Context awareness
- [x] Multi-tool support

### Phase 2
- [ ] Interactive multi-step workflows
- [ ] Learning from user feedback
- [ ] Custom tool configurations
- [ ] Team collaboration features

### Phase 3
- [ ] Multiple AI provider support (OpenAI, etc.)
- [ ] Advanced context analysis (logs, metrics)
- [ ] Automated fix suggestions
- [ ] Integration with monitoring tools

## 💡 Tips

1. **Be Specific**: "my terraform plan shows 5 resources changing" vs "terraform error"
2. **Use Troubleshoot Mode**: For complex issues, use `-troubleshoot` flag
3. **Check Context**: AI works better when you're in the right directory
4. **Review Commands**: Always review AI suggestions before confirming
5. **Provide Feedback**: Use GitHub issues to report AI accuracy problems

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Areas needing help:**
- New tool integrations
- AI prompt improvements
- Testing across different environments
- Documentation and examples
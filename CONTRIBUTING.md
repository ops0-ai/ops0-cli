# Contributing to ops0

Thank you for considering contributing to ops0! We welcome contributions from the community and are excited to work with you to make ops0 the best AI-powered DevOps CLI tool.

## 🚀 Quick Start

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/your-username/ops0-cli.git
   cd ops0-cli
   ```
3. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```
4. **Make your changes** and test them
5. **Submit a pull request**

## 📋 Ways to Contribute

### 🐛 Bug Reports
- Use the [issue tracker](https://github.com/ops0-ai/ops0-cli/issues) to report bugs
- Check if the issue already exists before creating a new one
- Include steps to reproduce, expected behavior, and actual behavior
- Add your environment details (OS, Go version, tool versions)

### 💡 Feature Requests
- Open an issue with the `enhancement` label
- Describe the feature and why it would be useful
- Include examples of how the feature would work
- Consider implementing it yourself if you're able!

### 🔧 Code Contributions
We welcome contributions in these areas:

#### **New Tool Support**
Add support for additional DevOps tools:
- Cloud providers (GCP, Azure, DigitalOcean)
- CI/CD tools (Jenkins, GitLab CI, GitHub Actions)
- Monitoring tools (Prometheus, Grafana)
- Container orchestration (Docker Swarm, Nomad)

#### **Better Natural Language Understanding**
- Improve regex patterns in `parseIntent()`
- Add more command variations
- Handle edge cases and typos

#### **AI Enhancements**
- Improve Claude prompts for better accuracy
- Add support for other AI providers (OpenAI, etc.)
- Enhance context gathering
- Better error handling for AI responses

#### **User Experience**
- Better error messages
- Improved help text
- Progress indicators for long-running commands
- Configuration file support

## 🛠️ Development Setup

### Prerequisites
- Go 1.19 or later
- Git
- (Optional) Claude API key for AI features

### Setup
```bash
# Clone the repository
git clone https://github.com/ops0-ai/ops0-cli.git
cd ops0-cli

# Build the project
go build -o ops0 main.go

# Run tests
go test ./...

# Test the CLI
./ops0 -m "test command"
```

### Development Workflow
```bash
# Make changes to main.go
vim main.go

# Build and test
go build -o ops0 main.go
./ops0 -m "your test command"

# Test AI mode (if you have API key)
export ANTHROPIC_API_KEY=your_key
./ops0 -m "test ai command" -ai

# Run any existing tests
go test -v
```

## 📝 Code Guidelines

### Go Style
- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions focused and small

### Adding New Tools
To add support for a new tool, you need to:

1. **Add regex patterns** in `parseIntent()`:
```go
// YourTool patterns
if matched, _ := regexp.MatchString(`your.*pattern`, input); matched {
    return &CommandSuggestion{
        Tool:        "yourtool",
        Command:     "yourtool command",
        Description: "This will do something with yourtool.",
        Intent:      "your intent description",
        Confidence:  0.8,
        AIGenerated: false,
    }
}
```

2. **Add installation command** in `getInstallCommand()`:
```go
case "yourtool":
    return "brew install yourtool"  // macOS
    // Add other OS support
```

3. **Update system context** in `gatherSystemContext()`:
```go
tools := []string{"terraform", "kubectl", "docker", "yourtool"}
```

4. **Add version detection** in `getToolVersion()` if needed

### Adding AI Enhancements
- Update the system prompt in `getAISuggestion()` to include your tool
- Add tool-specific troubleshooting patterns
- Test with various natural language inputs

### Testing Your Changes
```bash
# Test basic functionality
./ops0 -m "help with my new tool"

# Test edge cases
./ops0 -m "edge case input"

# Test AI mode
./ops0 -m "complex scenario" -ai

# Test troubleshooting
./ops0 -m "my new tool is broken" -troubleshoot
```

## 🧪 Testing

### Manual Testing
- Test your changes with real DevOps scenarios
- Try various natural language inputs
- Test both with and without AI mode
- Verify tool installation works
- Test on different operating systems if possible

### Adding Tests
We welcome automated tests! Consider adding:
- Unit tests for `parseIntent()` function
- Integration tests for AI responses
- End-to-end tests for complete workflows

```go
func TestParseIntent(t *testing.T) {
    suggestion := parseIntent("test input")
    if suggestion == nil {
        t.Error("Expected suggestion, got nil")
    }
    // Add more assertions
}
```

## 📖 Documentation

### Code Documentation
- Add comments to explain complex logic
- Document new functions and types
- Update existing comments if you change behavior

### User Documentation
- Update README.md if you add new features
- Add examples for new tools or capabilities
- Update help text in `showHelp()` function

## 🔄 Pull Request Process

### Before Submitting
- [ ] Code follows Go conventions
- [ ] Changes are tested manually
- [ ] Documentation is updated
- [ ] Commit messages are clear and descriptive

### Pull Request Template
When creating a PR, please include:

**What this PR does:**
- Brief description of changes

**Testing:**
- How you tested the changes
- Example commands that work

**Documentation:**
- What documentation was updated

**Breaking Changes:**
- Any breaking changes (hopefully none!)

### Example Commit Messages
```
feat: add kubectl support for namespace operations
fix: handle edge case in terraform pattern matching
docs: update README with new tool examples
refactor: simplify parseIntent function structure
```

## 🎯 Priority Areas

We're especially looking for contributions in:

### High Priority
- **Kubernetes enhancements** - Better kubectl support, Helm, operators
- **Cloud provider CLIs** - GCP, Azure, multi-cloud scenarios
- **Error handling** - Better error messages, recovery suggestions
- **Cross-platform support** - Windows, Linux installation improvements

### Medium Priority
- **CI/CD integrations** - GitHub Actions, GitLab CI, Jenkins
- **Monitoring tools** - Prometheus, Grafana, alerting
- **Configuration management** - Config file support, user preferences
- **Performance** - Faster response times, caching

### Nice to Have
- **Multiple AI providers** - OpenAI, local models
- **Plugin system** - External tool integrations
- **Web interface** - Browser-based command builder
- **Team features** - Shared configurations, collaboration

## 🤝 Community

### Getting Help
- Open an issue for questions
- Join discussions in existing issues
- Check the documentation first

### Code of Conduct
- Be respectful and inclusive
- Provide constructive feedback
- Help others learn and contribute
- Focus on the best outcome for the project

### Recognition
Contributors will be:
- Listed in the README
- Mentioned in release notes
- Invited to provide input on project direction

## 🏷️ Release Process

### Versioning
We use semantic versioning (semver):
- `v1.0.0` - Major release with breaking changes
- `v1.1.0` - Minor release with new features
- `v1.1.1` - Patch release with bug fixes

### Release Notes
Each release includes:
- New features added
- Bug fixes
- Breaking changes (if any)
- Contributors recognized

## 📞 Questions?

- **General questions**: Open an issue with the `question` label
- **Feature discussion**: Start a discussion on GitHub
- **Bug reports**: Use the bug report template
- **Security issues**: Contact maintainers directly

Thank you for contributing to ops0! Together we can make DevOps more accessible and intuitive for everyone. 🚀
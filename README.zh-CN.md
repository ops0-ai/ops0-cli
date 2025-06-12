<p align="center">
  <img src="assets/logo.jpg" alt="ops0 CLI Logo" width="150">
</p>

<p align="center">
  <a href="README.md">English</a> •
  <a href="README.zh-CN.md">简体中文</a> •
  <a href="README.de.md">Deutsch</a> •
  <a href="README.fr.md">Français</a> •
  <a href="README.es.md">Español</a> •
  <a href="README.pt-BR.md">Português</a>
</p>

---

<p align="center">
ops0 是一个智能命令行工具，可以将自然语言转换为 DevOps 命令。<br>
由 Claude AI 提供支持，它通过理解您的意图来简化复杂的 DevOps 任务，<br>
生成正确的命令，使 DevOps 管理更加便捷高效。
</p>

## ops0 演示

![ops0 CLI Demo](assets/ops0cli.gif)
*观看 ops0 如何将自然语言转换为强大的 DevOps 命令*

## 快速开始

### 安装
```bash
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | bash
```

### 基本用法
```bash
# 规则模式 (无需 API 密钥)
ops0 -m "i want to plan my iac code"

# AI 支持模式 (需要 API 密钥)
export ANTHROPIC_API_KEY=your_key_here
ops0 -m "check if my kubernetes pods are running" -ai

# 故障排除模式
ops0 -m "my terraform apply is failing with state lock" -troubleshoot
```

[继续添加其他章节的翻译...] 
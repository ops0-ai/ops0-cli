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
ops0 é uma ferramenta CLI inteligente que transforma linguagem natural em comandos DevOps.<br>
Alimentado por Claude AI, simplifica tarefas complexas de DevOps entendendo sua intenção<br>
e gerando os comandos corretos, tornando o gerenciamento DevOps mais acessível e eficiente.
</p>

## ops0 em Ação

![ops0 CLI Demo](assets/ops0cli.gif)
*Veja o ops0 traduzir linguagem natural em poderosos comandos DevOps*

## Início Rápido

### Instalação
```bash
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | bash
```

### Uso Básico
```bash
# Modo baseado em regras (não requer chave API)
ops0 -m "i want to plan my iac code"

# Modo IA (requer chave API)
export ANTHROPIC_API_KEY=your_key_here
ops0 -m "check if my kubernetes pods are running" -ai

# Modo de solução de problemas
ops0 -m "my terraform apply is failing with state lock" -troubleshoot
```

[Continuação da tradução das outras seções...] 
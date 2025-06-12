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
ops0 est un outil CLI intelligent qui transforme le langage naturel en commandes DevOps.<br>
Propulsé par Claude AI, il simplifie les tâches DevOps complexes en comprenant vos intentions<br>
et en générant les bonnes commandes, rendant la gestion DevOps plus accessible et efficace.
</p>

## ops0 en Action

![ops0 CLI Demo](assets/ops0cli.gif)
*Regardez ops0 traduire le langage naturel en puissantes commandes DevOps*

## Démarrage Rapide

### Installation
```bash
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | bash
```

### Utilisation de Base
```bash
# Mode basé sur des règles (pas de clé API nécessaire)
ops0 -m "i want to plan my iac code"

# Mode IA (clé API requise)
export ANTHROPIC_API_KEY=your_key_here
ops0 -m "check if my kubernetes pods are running" -ai

# Mode dépannage
ops0 -m "my terraform apply is failing with state lock" -troubleshoot
```

[Suite de la traduction des autres sections...] 
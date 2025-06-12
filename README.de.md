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
ops0 ist ein intelligentes CLI-Tool, das natürliche Sprache in DevOps-Befehle umwandelt.<br>
Unterstützt durch Claude AI vereinfacht es komplexe DevOps-Aufgaben, indem es Ihre Absicht versteht<br>
und die richtigen Befehle generiert, wodurch DevOps-Management zugänglicher und effizienter wird.
</p>

## ops0 in Aktion

![ops0 CLI Demo](assets/ops0cli.gif)
*Sehen Sie, wie ops0 natürliche Sprache in leistungsstarke DevOps-Befehle übersetzt*

## Schnellstart

### Installation
```bash
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | bash
```

### Grundlegende Verwendung
```bash
# Regelbasierter Modus (kein API-Schlüssel erforderlich)
ops0 -m "i want to plan my iac code"

# KI-gestützter Modus (API-Schlüssel erforderlich)
export ANTHROPIC_API_KEY=your_key_here
ops0 -m "check if my kubernetes pods are running" -ai

# Fehlerbehebungsmodus
ops0 -m "my terraform apply is failing with state lock" -troubleshoot
```

[Fortsetzung der Übersetzung weiterer Abschnitte...] 
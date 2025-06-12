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
ops0 es una herramienta CLI inteligente que transforma el lenguaje natural en comandos DevOps.<br>
Impulsado por Claude AI, simplifica las tareas complejas de DevOps entendiendo tu intención<br>
y generando los comandos correctos, haciendo la gestión DevOps más accesible y eficiente.
</p>

## ops0 en Acción

![ops0 CLI Demo](assets/ops0cli.gif)
*Mira cómo ops0 traduce el lenguaje natural en poderosos comandos DevOps*

## Inicio Rápido

### Instalación
```bash
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | bash
```

### Uso Básico
```bash
# Modo basado en reglas (no requiere clave API)
ops0 -m "i want to plan my iac code"

# Modo IA (requiere clave API)
export ANTHROPIC_API_KEY=your_key_here
ops0 -m "check if my kubernetes pods are running" -ai

# Modo de solución de problemas
ops0 -m "my terraform apply is failing with state lock" -troubleshoot
```

[Continuación de la traducción de otras secciones...] 
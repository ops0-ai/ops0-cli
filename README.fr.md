<p align="center">
  <img src="assets/logo.jpg" alt="ops0 CLI Logo" width="150">
</p>

<p align="center">
  <a href="./README.zh-CN.md">ReadMe in Chinese</a> • 
  <a href="./README.de.md">ReadMe in German</a> • 
  ReadMe in French • 
  <a href="./README.es.md">ReadMe in Spanish</a> • 
  <a href="./README.pt-BR.md">ReadMe in Portuguese</a> • 
  <a href="https://discord.gg/4vnuq2WJrV">Communauté Discord</a>
</p>

<p align="center">
  <a href="https://github.com/ops0-ai/ops0-cli/commits/main"><img src="https://img.shields.io/github/last-commit/ops0-ai/ops0-cli" alt="Last Commit"></a>
  <a href="https://github.com/ops0-ai/ops0-cli/releases"><img src="https://img.shields.io/github/v/release/ops0-ai/ops0-cli" alt="Latest Release"></a>
  <a href="https://github.com/ops0-ai/ops0-cli/stargazers"><img src="https://img.shields.io/github/stars/ops0-ai/ops0-cli" alt="GitHub Stars"></a>
  <a href="https://discord.gg/4vnuq2WJrV"><img src="https://img.shields.io/badge/Community-Discord-7289DA?logo=discord" alt="Discord"></a>
</p>

---

<p align="center">
ops0 est un outil CLI intelligent qui transforme le langage naturel en opérations DevOps.<br>
Propulsé par Claude AI, il simplifie les tâches DevOps complexes en comprenant vos intentions<br>
et en effectuant les bonnes opérations, rendant la gestion DevOps plus accessible et efficace.
</p>

## ops0 en Action

![ops0 CLI Demo](assets/ops0cli.gif)
*Regardez ops0 traduire le langage naturel en puissantes opérations DevOps*

### Installation
```bash
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | bash
```

### Utilisation de Base
```bash
# Mode interactif (par défaut) - lancez simplement ops0 !
ops0

# Mode basé sur des règles (pas de clé API nécessaire)
ops0 -m "je veux planifier mon code IaC"

# Mode IA (clé API requise)
export ANTHROPIC_API_KEY=your_key_here
ops0 -m "vérifier si mes pods kubernetes fonctionnent" -ai

# Mode dépannage
ops0 -m "mon terraform apply échoue à cause d'un state lock" -troubleshoot
```

*Lancez simplement `ops0` pour entrer en mode interactif et commencer à discuter avec des requêtes en langage naturel !*

## Opérations en Action

Voici des exemples concrets d'ops0 en action avec différents outils :

### Mode Administrateur Kafka
[![Vidéo du Mode Administrateur Kafka](assets/kafka-tn.png)](https://www.loom.com/share/c800f1f15865489780586c9d154ef365?sid=0e17ade7-6035-4eea-853a-c0e924ec4715)

*Exemple : Gérez interactivement les clusters Kafka en utilisant le langage naturel.*

## Mode Interactif

![ops0 Mode Interactif](assets/ops0-intro-cli.png)
*Entrez en mode interactif en lançant 'ops0' et commencez à discuter avec des commandes en langage naturel*

### Opérations AWS CLI
![AWS CLI Example](assets/aws.png)
*Exemple : Gestion des ressources AWS en langage naturel*

### Gestion des Conteneurs Docker
![Docker Example](assets/docker.png)
*Exemple : Gestion des conteneurs et images Docker en langage simple*

### Automatisation Ansible
![Ansible Example](assets/ansible.png)
![Ansible Playbook](assets/ansible-playbook.png)
*Exemple : Exécution et validation faciles des playbooks Ansible*

### Infrastructure Terraform
![Terraform Example](assets/terraform.png)
*Exemple : Gestion de l'infrastructure as code en langage naturel*

### Opérations Kubernetes
![Kubernetes Example](assets/kubernetes.png)
*Exemple : Gestion simplifiée des clusters Kubernetes et dépannage*

### Installer tous les outils en une seule commande

![CLI Installer Tous les Outils](assets/cli-install.png)

Vous pouvez désormais installer tous les outils DevOps supportés avec une seule commande :

```bash
ops0 --install
```

Cela installera automatiquement Terraform, Ansible, kubectl, Docker, Helm, AWS CLI, gcloud et Azure CLI, puis affichera leurs versions dans un tableau récapitulatif.

## Outils et Fonctionnalités Supportés

### Outils Principaux
- **Terraform** - Infrastructure as Code
- **Ansible** - Gestion de Configuration
- **Kubernetes (kubectl)** - Orchestration de Conteneurs
- **Docker** - Conteneurisation
- **AWS CLI** - Amazon Web Services
- **Helm** - Gestionnaire de Paquets Kubernetes
- **gcloud** - Google Cloud Platform
- **Azure CLI** - Microsoft Azure
- **System Admin** - Administration Système Linux

### Exemples d'Administration Système & Analyse de Logs
```bash
# Analyser les logs d'un pod Kubernetes et obtenir un résumé IA avec recommandations
ops0 -m "analyser les logs du pod my-app-123 dans le namespace prod"

# Analyser un fichier de log spécifique pour détecter des problèmes
ops0 -m "analyser /var/log/nginx/error.log"

# Surveiller les ressources système
ops0 -m "afficher l'utilisation de la mémoire sur ma machine"
ops0 -m "vérifier l'espace disque"
ops0 -m "afficher l'utilisation du CPU"

# Gérer les services système
ops0 -m "redémarrer le service nginx"
ops0 -m "vérifier l'état du service apache2"

# Gestion des paquets
ops0 -m "installer le paquet docker"
ops0 -m "mettre à jour les paquets système"

# Journaux système
ops0 -m "afficher les journaux système"
ops0 -m "vérifier les journaux journalctl"
```
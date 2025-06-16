<p align="center">
  <img src="assets/logo.jpg" alt="ops0 CLI Logo" width="150">
</p>

<p align="center">
  <a href="./README.zh-CN.md">ReadMe in Chinese</a> • 
  <a href="./README.de.md">ReadMe in German</a> • 
  ReadMe in French • 
  <a href="./README.es.md">ReadMe in Spanish</a> • 
  <a href="./README.pt-BR.md">ReadMe in Portuguese</a> • 
  <a href="https://join.slack.com/t/ops0/shared_invite/zt-37akwqb1v-BvfK7AioDlRhje94UN2tkw">Slack Community</a>
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

### Installer tous les outils en une seule commande

![CLI Installer Tous les Outils](assets/cli-install.png)

Vous pouvez désormais installer tous les outils DevOps supportés avec une seule commande :

```bash
ops0 --install
```

Cela installera automatiquement Terraform, Ansible, kubectl, Docker, Helm, AWS CLI, gcloud et Azure CLI, puis affichera leurs versions dans un tableau récapitulatif.

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

## Exemples de Commandes en Action

Voici des exemples concrets d'ops0 en action avec différents outils :

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

Vous pouvez désormais installer tous les outils DevOps supportés avec une seule commande :

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

### Fonctionnalités Clés
- Traduction en langage naturel
- Dépannage assisté par IA
- **Analyse des logs de pods Kubernetes avec résumé IA et suggestions de commandes**
- **Analyse de fichiers de logs pour détection de problèmes et contexte**
- Suggestions contextuelles
- Exécution sécurisée avec confirmation
- Support de simulation pour les opérations destructives
- Installation automatique des outils

## Mode IA vs Mode Règles

| Fonctionnalité | Mode Règles | Mode IA |
|---------|------------|---------|
| Configuration | Pas de clé API | Nécessite ANTHROPIC_API_KEY |
| Vitesse | Instantané | ~2-3 secondes |
| Compréhension | Correspondance de motifs | Langage naturel |
| Conscience contextuelle | Limitée | Élevée |
| Dépannage | Basique | Avancé |
| Scénarios complexes | Limité | Excellent |
| Utilisation hors ligne | ✅ | ❌ |

## Configuration

### Variables d'Environnement
```bash
# Requis pour les fonctionnalités IA
export ANTHROPIC_API_KEY=your_api_key

# Optionnel : Personnaliser le comportement de l'IA
export OPS0_AI_MODEL=claude-3-sonnet-20240229  # Modèle par défaut
export OPS0_MAX_TOKENS=1024                    # Longueur de réponse
```

## Confidentialité et Sécurité

- **Clé API** : Stockée localement comme variable d'environnement
- **Pas de Stockage de Données** : Les commandes et le contexte ne sont pas stockés par ops0
- **Confidentialité Anthropic** : Suit les politiques de traitement des données d'Anthropic
- **Traitement Local** : Le mode règles fonctionne entièrement hors ligne

## Feuille de Route

### Actuel
- [x] Intégration Claude AI
- [x] Mode de dépannage basique
- [x] Conscience contextuelle
- [x] Support multi-outils

### À Venir
- [ ] Modèle personnalisé pour environnements hors ligne
- [ ] Workflows interactifs multi-étapes
- [ ] Apprentissage à partir des retours utilisateurs
- [ ] Configurations d'outils personnalisées
- [ ] Support de multiples fournisseurs IA
- [ ] Analyse contextuelle avancée
- [ ] Fonctionnalités de collaboration d'équipe

## Conseils

1. **Soyez Spécifique** : "Mon plan terraform montre 5 ressources modifiées" vs "erreur terraform"
2. **Utilisez le Mode Dépannage** : Pour les problèmes complexes, utilisez le flag `-troubleshoot`
3. **Vérifiez le Contexte** : L'IA fonctionne mieux dans le bon répertoire
4. **Vérifiez les Commandes** : Toujours examiner les suggestions de l'IA avant confirmation
5. **Donnez des Retours** : Utilisez GitHub Issues pour signaler les problèmes de précision de l'IA 
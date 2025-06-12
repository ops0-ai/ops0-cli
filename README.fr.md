<p align="center">
  <img src="assets/logo.jpg" alt="ops0 CLI Logo" width="150">
</p>
<p align="center">
[ReadMe in Chinese](./README.zh-CN.md) • [ReadMe in German](./README.de.md) • ReadMe in French • [ReadMe in Spanish](./README.es.md) • [ReadMe in Portuguese](./README.pt-BR.md) • [Slack Community](https://join.slack.com/t/ops0/shared_invite/zt-37akwqb1v-BvfK7AioDlRhje94UN2tkw)
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
*Exemple : Exécution et validation faciles des playbooks Ansible*

### Infrastructure Terraform
![Terraform Example](assets/terraform.png)
*Exemple : Gestion de l'infrastructure as code en langage naturel*

### Opérations Kubernetes
![Kubernetes Example](assets/kubernetes.png)
*Exemple : Gestion simplifiée des clusters Kubernetes et dépannage*

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

### Fonctionnalités Clés
- Traduction en langage naturel
- Dépannage assisté par IA
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
<p align="center">
  <img src="assets/logo.jpg" alt="ops0 CLI Logo" width="150">
</p>

<p align="center">
  <a href="./README.zh-CN.md">ReadMe in Chinese</a> • 
  <a href="./README.de.md">ReadMe in German</a> • 
  <a href="./README.fr.md">ReadMe in French</a> • 
  ReadMe in Spanish • 
  <a href="./README.pt-BR.md">ReadMe in Portuguese</a> • 
  <a href="https://discord.gg/4vnuq2WJrV">Comunidad de Discord</a>
</p>

<p align="center">
  <a href="https://github.com/ops0-ai/ops0-cli/commits/main"><img src="https://img.shields.io/github/last-commit/ops0-ai/ops0-cli" alt="Last Commit"></a>
  <a href="https://github.com/ops0-ai/ops0-cli/releases"><img src="https://img.shields.io/github/v/release/ops0-ai/ops0-cli" alt="Latest Release"></a>
  <a href="https://github.com/ops0-ai/ops0-cli/stargazers"><img src="https://img.shields.io/github/stars/ops0-ai/ops0-cli" alt="GitHub Stars"></a>
  <a href="https://discord.gg/4vnuq2WJrV"><img src="https://img.shields.io/badge/Community-Discord-7289DA?logo=discord" alt="Discord"></a>
</p>

---

<p align="center">
ops0 es una herramienta CLI inteligente que transforma el lenguaje natural en operaciones DevOps.<br>
Impulsado por Claude AI, simplifica las tareas complejas de DevOps entendiendo tu intención<br>
y realizando las operaciones correctas, haciendo la gestión DevOps más accesible y eficiente.
</p>

## ops0 en Acción

![ops0 CLI Demo](assets/ops0cli.gif)
*Mira cómo ops0 traduce el lenguaje natural en poderosos comandos DevOps*

### Instalación
```bash
curl -fsSL https://raw.githubusercontent.com/ops0-ai/ops0-cli/main/install.sh | bash
```

### Uso Básico
```bash
# Modo interactivo (por defecto) - ¡simplemente ejecuta ops0!
ops0

# Modo basado en reglas (no requiere clave API)
ops0 -m "quiero planificar mi código IaC"

# Modo IA (requiere clave API)
export ANTHROPIC_API_KEY=your_key_here
ops0 -m "verifica si mis pods de kubernetes están corriendo" -ai

# Modo de solución de problemas
ops0 -m "mi terraform apply falla por state lock" -troubleshoot
```

*¡Simplemente ejecuta `ops0` para entrar en modo interactivo y comenzar a chatear con solicitudes en lenguaje natural!*

## Operaciones en Acción

Aquí hay algunos ejemplos reales de ops0 en acción con diferentes herramientas:

### Modo Administrador de Kafka
[![Video del Modo Administrador de Kafka](assets/kafka-tn.png)](https://www.loom.com/share/c800f1f15865489780586c9d154ef365?sid=0e17ade7-6035-4eea-853a-c0e924ec4715)

*Ejemplo: Administre interactivamente los clústeres de Kafka utilizando lenguaje natural.*

## Modo Interactivo

![ops0 Modo Interactivo](assets/ops0-intro-cli.png)
*Entra en modo interactivo ejecutando 'ops0' y comienza a chatear con comandos en lenguaje natural*

### Operaciones de AWS CLI
![AWS CLI Example](assets/aws.png)
*Ejemplo: Gestión de recursos AWS usando lenguaje natural*

### Gestión de Contenedores Docker
![Docker Example](assets/docker.png)
*Ejemplo: Gestión de contenedores e imágenes Docker con lenguaje simple*

### Automatización con Ansible
![Ansible Example](assets/ansible.png)
![Ansible Playbook](assets/ansible-playbook.png)
*Ejemplo: Ejecución y validación sencilla de playbooks de Ansible*

### Infraestructura con Terraform
![Terraform Example](assets/terraform.png)
*Ejemplo: Gestión de infraestructura como código con lenguaje natural*

### Operaciones de Kubernetes
![Kubernetes Example](assets/kubernetes.png)
*Ejemplo: Gestión simplificada de clusters Kubernetes y solución de problemas*

### Instala todas las herramientas con un solo comando

![CLI Instalar Todas las Herramientas](assets/cli-install.png)

Ahora puedes instalar todas las herramientas DevOps soportadas con un solo comando:

```bash
ops0 --install
```

Esto instalará automáticamente Terraform, Ansible, kubectl, Docker, Helm, AWS CLI, gcloud y Azure CLI, y mostrará sus versiones en una tabla resumen.

## Herramientas y Funcionalidades Soportadas

### Herramientas Principales
- **Terraform** - Infraestructura como Código
- **Ansible** - Gestión de Configuración
- **Kubernetes (kubectl)** - Orquestación de Contenedores
- **Docker** - Contenedorización
- **AWS CLI** - Amazon Web Services
- **Helm** - Gestor de Paquetes Kubernetes
- **gcloud** - Google Cloud Platform
- **Azure CLI** - Microsoft Azure
- **System Admin** - Administración de Sistemas Linux

### Ejemplos de Administración de Sistema y Análisis de Logs
```bash
# Monitorear recursos del sistema
ops0 -m "mostrar uso de memoria en mi máquina"
ops0 -m "verificar espacio en disco"
ops0 -m "mostrar uso de CPU"

# Gestionar servicios del sistema
ops0 -m "reiniciar servicio nginx"
ops0 -m "verificar estado del servicio apache2"

# Gestión de paquetes
ops0 -m "instalar paquete docker"
ops0 -m "actualizar paquetes del sistema"

# Registros del sistema
ops0 -m "mostrar registros del sistema"
ops0 -m "verificar registros journalctl"

# Analizar logs de un pod de Kubernetes y obtener resumen IA con recomendaciones
ops0 -m "analizar logs del pod my-app-123 en el namespace prod"

# Analizar un archivo de log específico en busca de problemas
ops0 -m "analizar /var/log/nginx/error.log"
```

### Características Principales
- Traducción de lenguaje natural
- Solución de problemas asistida por IA
- **Análisis de logs de pods de Kubernetes con resumen IA y comandos sugeridos**
- **Análisis de archivos de log para problemas y contexto**
- Sugerencias contextuales
- Ejecución segura con confirmación
- Soporte de simulación para operaciones destructivas
- Instalación automática de herramientas

## Modo IA vs Modo Reglas

| Característica | Modo Reglas | Modo IA |
|---------|------------|---------|
| Configuración | No requiere clave API | Requiere ANTHROPIC_API_KEY |
| Velocidad | Instantáneo | ~2-3 segundos |
| Comprensión | Coincidencia de patrones | Lenguaje natural |
| Conciencia contextual | Limitada | Alta |
| Solución de problemas | Básica | Avanzada |
| Escenarios complejos | Limitado | Excelente |
| Uso sin conexión | ✅ | ❌ |

## Configuración

### Variables de Entorno
```bash
# Requerido para funciones de IA
export ANTHROPIC_API_KEY=your_api_key

# Opcional: Personalizar comportamiento de IA
export OPS0_AI_MODEL=claude-3-sonnet-20240229  # Modelo predeterminado
export OPS0_MAX_TOKENS=1024                    # Longitud de respuesta
```

## Privacidad y Seguridad

- **Clave API**: Almacenada localmente como variable de entorno
- **Sin Almacenamiento de Datos**: ops0 no almacena comandos ni contexto
- **Privacidad de Anthropic**: Sigue las políticas de tratamiento de datos de Anthropic
- **Procesamiento Local**: El modo reglas funciona completamente sin conexión

## Hoja de Ruta

### Actual
- [x] Integración con Claude AI
- [x] Modo básico de solución de problemas
- [x] Conciencia contextual
- [x] Soporte multi-herramienta

### Próximamente
- [ ] Modelo personalizado para entornos sin conexión
- [ ] Flujos de trabajo interactivos multi-paso
- [ ] Aprendizaje de retroalimentación del usuario
- [ ] Configuraciones personalizadas de herramientas
- [ ] Soporte para múltiples proveedores de IA
- [ ] Análisis contextual avanzado
- [ ] Funciones de colaboración en equipo

## Consejos

1. **Sé Específico**: "Mi plan de terraform muestra 5 recursos cambiando" vs "error de terraform"
2. **Usa el Modo de Solución de Problemas**: Para problemas complejos, usa la bandera `-troubleshoot`
3. **Verifica el Contexto**: La IA funciona mejor en el directorio correcto
4. **Revisa los Comandos**: Siempre revisa las sugerencias de IA antes de confirmar
5. **Proporciona Retroalimentación**: Usa GitHub Issues para reportar problemas de precisión de IA 
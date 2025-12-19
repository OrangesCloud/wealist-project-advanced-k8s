#!/bin/bash
# =============================================================================
# Service Chart Generator - Creates production-ready Helm charts for Go services
# =============================================================================

set -e

SERVICE_NAME=$1
SERVICE_PORT=$2
HAS_DB=${3:-true}  # Default to true

if [ -z "$SERVICE_NAME" ] || [ -z "$SERVICE_PORT" ]; then
  echo "Usage: $0 <service-name> <service-port> [has-db]"
  echo "Example: $0 board-service 8000 true"
  exit 1
fi

CHART_DIR="k8s/helm/charts/${SERVICE_NAME}"
TEMPLATES_DIR="${CHART_DIR}/templates"

echo "📦 Generating production-ready chart for ${SERVICE_NAME}..."

# Create templates directory
mkdir -p "${TEMPLATES_DIR}"

# Create template files (all use wealist-common)
cat > "${TEMPLATES_DIR}/deployment.yaml" <<EOF
{{- include "wealist-common.deployment" . }}
EOF

cat > "${TEMPLATES_DIR}/service.yaml" <<EOF
{{- include "wealist-common.service" . }}
EOF

cat > "${TEMPLATES_DIR}/configmap.yaml" <<EOF
{{- include "wealist-common.configmap" . }}
EOF

cat > "${TEMPLATES_DIR}/hpa.yaml" <<EOF
{{- include "wealist-common.hpa" . }}
EOF

cat > "${TEMPLATES_DIR}/serviceaccount.yaml" <<EOF
{{- include "wealist-common.serviceAccount" . }}
EOF

cat > "${TEMPLATES_DIR}/poddisruptionbudget.yaml" <<'EOF'
{{- if .Values.podDisruptionBudget }}
{{- if .Values.podDisruptionBudget.enabled }}
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{ include "wealist-common.fullname" . }}
  labels:
    {{- include "wealist-common.labels" . | nindent 4 }}
spec:
  {{- if .Values.podDisruptionBudget.minAvailable }}
  minAvailable: {{ .Values.podDisruptionBudget.minAvailable }}
  {{- end }}
  {{- if .Values.podDisruptionBudget.maxUnavailable }}
  maxUnavailable: {{ .Values.podDisruptionBudget.maxUnavailable }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "wealist-common.selectorLabels" . | nindent 6 }}
{{- end }}
{{- end }}
EOF

cat > "${TEMPLATES_DIR}/networkpolicy.yaml" <<EOF
{{- include "wealist-common.networkPolicy" . }}
EOF

echo "✅ Templates created"

# Create values.yaml (production-ready baseline)
cat > "${CHART_DIR}/values.yaml" <<EOF
# =============================================================================
# ${SERVICE_NAME} - Production-Ready Configuration
# =============================================================================

# -----------------------------------------------------------------------------
# Global Configuration
# -----------------------------------------------------------------------------
global:
  namespace: wealist
  environment: production
  domain: wealist.co.kr
  imageRegistry: ""

# -----------------------------------------------------------------------------
# Image Configuration
# -----------------------------------------------------------------------------
image:
  repository: ${SERVICE_NAME}
  pullPolicy: IfNotPresent
  tag: "1.0.0"

imagePullSecrets: []

# -----------------------------------------------------------------------------
# Deployment Configuration
# -----------------------------------------------------------------------------
replicaCount: 2

strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1
    maxUnavailable: 0

# -----------------------------------------------------------------------------
# Service Configuration
# -----------------------------------------------------------------------------
service:
  type: ClusterIP
  port: ${SERVICE_PORT}
  targetPort: ${SERVICE_PORT}
  annotations: {}

# -----------------------------------------------------------------------------
# Application Configuration
# -----------------------------------------------------------------------------
config:
  PORT: "${SERVICE_PORT}"
  SERVER_PORT: "${SERVICE_PORT}"
  ENV: "production"
  LOG_LEVEL: "info"
EOF

# Add database config if service uses DB
if [ "$HAS_DB" = "true" ]; then
  cat >> "${CHART_DIR}/values.yaml" <<EOF

  # Database configuration
  DB_HOST: "postgres"
  DB_PORT: "5432"
  DB_NAME: "wealist_${SERVICE_NAME//-/_}_db"
  DB_SSL_MODE: "require"
EOF
fi

# Add common configs
cat >> "${CHART_DIR}/values.yaml" <<'EOF'

  # Redis configuration
  REDIS_HOST: "redis"
  REDIS_PORT: "6379"

  # S3 configuration
  S3_ENDPOINT: "http://minio:9000"
  S3_PUBLIC_ENDPOINT: "https://wealist.co.kr/minio"
  S3_BUCKET: "wealist-files"
  S3_REGION: "ap-northeast-2"

  # Service URLs
  AUTH_SERVICE_URL: "http://auth-service:8080"
  USER_SERVICE_URL: "http://user-service:8081"

# -----------------------------------------------------------------------------
# External Secrets
# -----------------------------------------------------------------------------
externalSecrets:
  enabled: false
  secretStore: aws-secrets-manager
  secretStoreKind: ClusterSecretStore
  refreshInterval: 1h

# -----------------------------------------------------------------------------
# Health Check Configuration
# -----------------------------------------------------------------------------
healthCheck:
  liveness:
    path: /health/live
    initialDelaySeconds: 30
    periodSeconds: 10
    timeoutSeconds: 5
    successThreshold: 1
    failureThreshold: 3
  readiness:
    path: /health/ready
    initialDelaySeconds: 20
    periodSeconds: 5
    timeoutSeconds: 3
    successThreshold: 1
    failureThreshold: 3

# -----------------------------------------------------------------------------
# Resource Management
# -----------------------------------------------------------------------------
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"

# -----------------------------------------------------------------------------
# Autoscaling
# -----------------------------------------------------------------------------
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 50
          periodSeconds: 60
        - type: Pods
          value: 1
          periodSeconds: 60
      selectPolicy: Min
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
        - type: Percent
          value: 100
          periodSeconds: 30
        - type: Pods
          value: 2
          periodSeconds: 30
      selectPolicy: Max

# -----------------------------------------------------------------------------
# Security
# -----------------------------------------------------------------------------
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000
  seccompProfile:
    type: RuntimeDefault

securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  readOnlyRootFilesystem: false
  runAsNonRoot: true
  runAsUser: 1000

# -----------------------------------------------------------------------------
# Service Account
# -----------------------------------------------------------------------------
serviceAccount:
  create: true
  annotations: {}
  name: ""
  automount: true

# -----------------------------------------------------------------------------
# Pod Disruption Budget
# -----------------------------------------------------------------------------
podDisruptionBudget:
  enabled: true
  minAvailable: 1

# -----------------------------------------------------------------------------
# Network Policy
# -----------------------------------------------------------------------------
networkPolicy:
  enabled: false

# -----------------------------------------------------------------------------
# Monitoring
# -----------------------------------------------------------------------------
metrics:
  enabled: true
  path: /metrics

podAnnotations: {}

nodeSelector: {}
tolerations: []
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchExpressions:
              - key: app.kubernetes.io/name
                operator: In
                values:
                  - ${SERVICE_NAME}
          topologyKey: kubernetes.io/hostname

extraEnv: []
extraEnvFrom: []
extraVolumes: []
extraVolumeMounts: []
EOF

echo "✅ values.yaml created"

# Create development overrides
cat > "${CHART_DIR}/values-develop-registry-local.yaml" <<EOF
# =============================================================================
# ${SERVICE_NAME} - Development Environment
# =============================================================================

global:
  namespace: wealist-dev
  environment: develop
  domain: local.wealist.co.kr
  imageRegistry: localhost:5001

image:
  pullPolicy: Always

replicaCount: 1

config:
  ENV: "dev"
  LOG_LEVEL: "debug"
  S3_PUBLIC_ENDPOINT: "https://local.wealist.co.kr/minio"
EOF

# Add DB SSL mode for dev
if [ "$HAS_DB" = "true" ]; then
  cat >> "${CHART_DIR}/values-develop-registry-local.yaml" <<EOF
  DB_SSL_MODE: "disable"
EOF
fi

cat >> "${CHART_DIR}/values-develop-registry-local.yaml" <<'EOF'

healthCheck:
  liveness:
    initialDelaySeconds: 15
  readiness:
    initialDelaySeconds: 5

resources:
  requests:
    memory: "128Mi"
    cpu: "50m"
  limits:
    memory: "256Mi"
    cpu: "200m"

autoscaling:
  enabled: false

podSecurityContext:
  runAsNonRoot: false

securityContext:
  allowPrivilegeEscalation: false
  runAsNonRoot: false

serviceAccount:
  create: false

networkPolicy:
  enabled: false

metrics:
  enabled: true

podDisruptionBudget:
  enabled: false
EOF

echo "✅ values-develop-registry-local.yaml created"

# Update dependencies
cd "${CHART_DIR}" && helm dependency update && cd -

echo "🎉 ${SERVICE_NAME} chart generated successfully!"
echo "📝 Next steps:"
echo "   - Review and customize ${CHART_DIR}/values.yaml"
echo "   - Add service-specific configuration"
echo "   - Run: helm lint ./k8s/helm/charts/${SERVICE_NAME}"

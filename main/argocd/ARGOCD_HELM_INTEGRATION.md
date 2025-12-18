# ArgoCD + Helm Integration Guide

## 🎉 Migration Complete!

All weAlist ArgoCD Applications have been successfully migrated from **Kustomize** to **Helm** source.

**Validation Results**: ✅ **72/72 tests passed**

---

## 📊 What Changed

### Before (Kustomize)
```yaml
spec:
  source:
    path: services/user-service/k8s/overlays/local
```

### After (Helm)
```yaml
spec:
  source:
    path: helm/charts/user-service
    helm:
      valueFiles:
        - values.yaml
        - values-develop-registry-local.yaml
      parameters:
        - name: image.tag
          value: "latest"
```

---

## 📦 Applications Overview

### Infrastructure (1)
- **wealist-infrastructure** - PostgreSQL, Redis, MinIO, shared ConfigMap, Ingress

### Services (8)
- **auth-service** (Spring Boot, port 8080)
- **user-service** (Go, port 8081)
- **board-service** (Go, port 8000)
- **chat-service** (Go, port 8001)
- **noti-service** (Go, port 8002)
- **storage-service** (Go, port 8003) - **NEW**
- **video-service** (Go, port 8004) - **NEW**
- **frontend** (React + Vite + NGINX, port 3000)

### Total: 9 Helm-based Applications

---

## 🚀 Deployment Guide

### Prerequisites

1. **ArgoCD Installed**
   ```bash
   kubectl create namespace argocd
   kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
   ```

2. **ArgoCD CLI Installed**
   ```bash
   # macOS
   brew install argocd

   # Linux
   curl -sSL -o argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
   chmod +x argocd
   sudo mv argocd /usr/local/bin/
   ```

3. **Access ArgoCD UI**
   ```bash
   # Port forward
   kubectl port-forward svc/argocd-server -n argocd 8080:443

   # Get initial admin password
   kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d

   # Login
   argocd login localhost:8080
   ```

---

## 🔧 Initial Setup

### Step 1: Create wealist Project

```bash
argocd proj create wealist \
  --description "weAlist Microservices Platform" \
  --dest https://kubernetes.default.svc,wealist-dev \
  --src https://github.com/your-org/wealist-project-advanced.git
```

Or via YAML (`argocd/apps/project.yaml`):
```bash
kubectl apply -f argocd/apps/project.yaml
```

### Step 2: Deploy Root Application (App of Apps Pattern)

```bash
kubectl apply -f argocd/apps/root-app.yaml
```

This will automatically deploy all 9 Applications:
- wealist-infrastructure
- auth-service
- user-service
- board-service
- chat-service
- noti-service
- storage-service
- video-service
- frontend

### Step 3: Monitor Deployment

```bash
# List all applications
argocd app list

# Watch specific app
argocd app get user-service --refresh

# View sync status
argocd app sync-status user-service

# View logs
argocd app logs user-service --follow
```

---

## 🔄 Update Workflows

### Update a Single Service

#### Option 1: ArgoCD Auto-sync (Recommended)
1. Update Helm chart values or templates
2. Commit and push to repository
3. ArgoCD automatically detects changes and syncs

#### Option 2: Manual Sync
```bash
# Via CLI
argocd app sync user-service

# Via UI
# Navigate to user-service → SYNC → SYNCHRONIZE
```

### Update Image Tag

#### Option 1: Helm Parameters (Quick)
```bash
argocd app set user-service \
  -p image.tag=v1.2.3
```

#### Option 2: Update Values File (Persistent)
```yaml
# helm/charts/user-service/values-develop-registry-local.yaml
image:
  tag: "v1.2.3"
```
Commit, push, and ArgoCD auto-syncs.

### Update Configuration

```yaml
# helm/charts/user-service/values-develop-registry-local.yaml
config:
  LOG_LEVEL: "info"  # Change from "debug"
  NEW_FEATURE_FLAG: "true"
```
Commit and push → Auto-sync.

---

## 🛠️ Advanced Operations

### Helm Parameters Override

```bash
# Override multiple parameters
argocd app set user-service \
  -p image.tag=latest \
  -p replicaCount=3 \
  -p config.LOG_LEVEL=debug
```

### Use Different Values File

Edit Application manifest:
```yaml
# argocd/apps/user-service.yaml
spec:
  source:
    helm:
      valueFiles:
        - values.yaml
        - values-production.yaml  # Changed from values-develop-registry-local.yaml
```

### Disable Auto-sync (Manual Mode)

```bash
argocd app set user-service --sync-policy none
```

Or edit Application:
```yaml
spec:
  syncPolicy: {}  # Remove automated section
```

### Re-enable Auto-sync

```bash
argocd app set user-service \
  --sync-policy automated \
  --auto-prune \
  --self-heal
```

---

## 🔍 Troubleshooting

### Application Out of Sync

```bash
# Check diff
argocd app diff user-service

# Force refresh
argocd app get user-service --refresh --hard-refresh

# Manual sync
argocd app sync user-service
```

### Helm Template Errors

```bash
# View rendered manifests
argocd app manifests user-service

# Check for template errors
helm template user-service ./helm/charts/user-service \
  -f ./helm/charts/user-service/values-develop-registry-local.yaml \
  --debug
```

### Sync Fails Due to Validation

```bash
# Skip validation (use with caution)
argocd app sync user-service --validate=false

# Or add to Application:
spec:
  syncPolicy:
    syncOptions:
      - Validate=false
```

### Prune Issues

```bash
# Sync without pruning
argocd app sync user-service --prune=false

# View what would be pruned
argocd app diff user-service
```

### Rollback

```bash
# View history
argocd app history user-service

# Rollback to previous version
argocd app rollback user-service

# Rollback to specific revision
argocd app rollback user-service 3
```

---

## 📋 Validation

### Validate All Applications

```bash
./argocd/scripts/validate-applications.sh
```

**Expected Output**:
```
✓ All tests passed!
ArgoCD Applications are ready for Helm! 🎉

Total Tests:  72
Passed:       72
Failed:       0
```

### Validate Individual Application

```bash
# Check YAML syntax
argocd app lint user-service

# Check Application health
argocd app get user-service

# Check sync status
argocd app sync-status user-service
```

---

## 🎯 Best Practices

### 1. GitOps Workflow
- ✅ **DO**: Make changes in Git, let ArgoCD sync
- ❌ **DON'T**: Apply manifests directly with `kubectl`

### 2. Values Management
- ✅ **DO**: Use environment-specific values files
- ✅ **DO**: Keep production values in separate branch/repo
- ❌ **DON'T**: Hardcode secrets in values files

### 3. Sync Policy
- ✅ **DO**: Enable auto-sync for development
- ✅ **DO**: Use manual sync for production (with approval)
- ✅ **DO**: Enable `prune: true` and `selfHeal: true`

### 4. Monitoring
- ✅ **DO**: Set up ArgoCD notifications (Slack, email)
- ✅ **DO**: Monitor sync status regularly
- ✅ **DO**: Use ArgoCD webhooks for faster sync

### 5. Helm Parameters
- ✅ **DO**: Use for temporary overrides (testing)
- ❌ **DON'T**: Use for permanent configuration (use values files)

---

## 🔐 Security Considerations

### Secrets Management

**Option 1: External Secrets Operator** (Recommended for production)
```yaml
# helm/charts/user-service/values-production.yaml
externalSecrets:
  enabled: true
  secretStore: aws-secrets-manager
```

**Option 2: Sealed Secrets**
```bash
# Create sealed secret
kubeseal --format yaml < secret.yaml > sealed-secret.yaml
```

**Option 3: ArgoCD Vault Plugin**
```yaml
spec:
  source:
    plugin:
      name: argocd-vault-plugin
```

### Repository Access

```bash
# Add private repository
argocd repo add https://github.com/your-org/wealist-private.git \
  --username your-username \
  --password your-token

# Or use SSH
argocd repo add git@github.com:your-org/wealist-private.git \
  --ssh-private-key-path ~/.ssh/id_rsa
```

---

## 📊 Monitoring & Observability

### ArgoCD Metrics (Prometheus)

ArgoCD exposes metrics on port 8082:
```bash
kubectl port-forward svc/argocd-metrics -n argocd 8082:8082
curl http://localhost:8082/metrics
```

### Application Health

```bash
# Check all apps health
argocd app list -o json | jq '.[] | {name: .metadata.name, health: .status.health.status}'

# Degraded apps only
argocd app list -o json | jq '.[] | select(.status.health.status != "Healthy")'
```

### Sync Waves

For controlled deployment order, use annotations:
```yaml
# helm/charts/wealist-infrastructure/templates/postgres/statefulset.yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "0"  # Deploy first

# helm/charts/user-service/templates/deployment.yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"  # Deploy after infrastructure
```

---

## 🔄 Migration Checklist

- [x] Update infrastructure Application to Helm
- [x] Update auth-service Application to Helm
- [x] Update user-service Application to Helm
- [x] Update board-service Application to Helm
- [x] Update chat-service Application to Helm
- [x] Update noti-service Application to Helm
- [x] Create storage-service Application
- [x] Create video-service Application
- [x] Update frontend Application to Helm
- [x] Validate all Applications (72 tests passed)
- [x] Create validation script
- [x] Document ArgoCD integration

---

## 📚 Additional Resources

- **ArgoCD Official Docs**: https://argo-cd.readthedocs.io/
- **Helm Integration**: https://argo-cd.readthedocs.io/en/stable/user-guide/helm/
- **Best Practices**: https://argo-cd.readthedocs.io/en/stable/user-guide/best_practices/
- **weAlist Helm Charts**: `../helm/PRODUCTION_READY_SUMMARY.md`
- **weAlist Quick Start**: `../helm/QUICK_START.md`

---

## 🆘 Support

### Common Issues

1. **"Application not found"**
   - Ensure root-app is deployed: `kubectl get app -n argocd`
   - Verify project exists: `argocd proj get wealist`

2. **"Repository not accessible"**
   - Check repo credentials: `argocd repo list`
   - Verify GitHub access token

3. **"Helm chart not found"**
   - Ensure charts exist in repository
   - Check chart path in Application spec
   - Run validation: `./argocd/scripts/validate-applications.sh`

4. **"Values file not found"**
   - Verify file exists: `ls helm/charts/user-service/values*.yaml`
   - Check valueFiles path in Application spec

---

**Status**: ✅ Phase 6 Complete - ArgoCD Integration Ready!
**Validation**: ✅ 72/72 tests passed
**Next Phase**: SonarQube Integration (Docker Compose only)

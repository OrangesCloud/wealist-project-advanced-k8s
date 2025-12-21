#!/bin/bash
# =============================================================================
# ArgoCD Applications Validation Script
# =============================================================================
# Validates all ArgoCD Application manifests for Helm integration

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APPS_DIR="$(cd "${SCRIPT_DIR}/../apps" && pwd)"
CHARTS_DIR="$(cd "${SCRIPT_DIR}/../../helm/charts" && pwd)"

echo "🔍 ArgoCD Applications Validation"
echo "========================================"
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

total_tests=0
passed_tests=0
failed_tests=0

# Function to print test result
print_result() {
  local test_name=$1
  local result=$2
  total_tests=$((total_tests + 1))

  if [ "$result" -eq 0 ]; then
    echo -e "${GREEN}✓${NC} $test_name"
    passed_tests=$((passed_tests + 1))
  else
    echo -e "${RED}✗${NC} $test_name"
    failed_tests=$((failed_tests + 1))
  fi
}

# Applications that should use Helm
HELM_APPS=(
  "wealist-infrastructure"
  "auth-service"
  "user-service"
  "board-service"
  "chat-service"
  "noti-service"
  "storage-service"
  "video-service"
  "frontend"
)

# Test 1: YAML Syntax
echo "📋 Test 1: YAML Syntax Validation"
echo "----------------------------------------"
for app in "${HELM_APPS[@]}"; do
  app_file="${APPS_DIR}/${app}.yaml"

  # Map infrastructure app name
  if [ "$app" = "wealist-infrastructure" ]; then
    app_file="${APPS_DIR}/infrastructure.yaml"
  fi

  if [ ! -f "$app_file" ]; then
    print_result "${app} YAML file exists" 1
    continue
  fi

  # Validate YAML syntax
  if python3 -c "import yaml; yaml.safe_load(open('$app_file'))" 2>/dev/null; then
    print_result "${app} YAML syntax" 0
  else
    print_result "${app} YAML syntax" 1
  fi
done
echo ""

# Test 2: Helm Source Configuration
echo "🎯 Test 2: Helm Source Configuration"
echo "----------------------------------------"
for app in "${HELM_APPS[@]}"; do
  app_file="${APPS_DIR}/${app}.yaml"

  if [ "$app" = "wealist-infrastructure" ]; then
    app_file="${APPS_DIR}/infrastructure.yaml"
  fi

  if [ ! -f "$app_file" ]; then
    continue
  fi

  # Check for helm path
  if grep -q "path: k8s/helm/charts/" "$app_file"; then
    print_result "${app} uses Helm path" 0
  else
    print_result "${app} uses Helm path" 1
  fi
done
echo ""

# Test 3: ValueFiles Configuration
echo "📝 Test 3: ValueFiles Present"
echo "----------------------------------------"
for app in "${HELM_APPS[@]}"; do
  app_file="${APPS_DIR}/${app}.yaml"

  if [ "$app" = "wealist-infrastructure" ]; then
    app_file="${APPS_DIR}/infrastructure.yaml"
  fi

  if [ ! -f "$app_file" ]; then
    continue
  fi

  # Check for valueFiles
  if grep -q "valueFiles:" "$app_file" && \
     grep -q "values.yaml" "$app_file" && \
     grep -q "values-develop-registry-local.yaml" "$app_file"; then
    print_result "${app} valueFiles configured" 0
  else
    print_result "${app} valueFiles configured" 1
  fi
done
echo ""

# Test 4: Chart Directory Exists
echo "📦 Test 4: Corresponding Helm Charts Exist"
echo "----------------------------------------"
for app in "${HELM_APPS[@]}"; do
  chart_dir="${CHARTS_DIR}/${app}"

  if [ "$app" = "wealist-infrastructure" ]; then
    chart_dir="${CHARTS_DIR}/wealist-infrastructure"
  fi

  if [ -d "$chart_dir" ] && [ -f "${chart_dir}/Chart.yaml" ]; then
    print_result "${app} chart exists" 0
  else
    print_result "${app} chart exists" 1
  fi
done
echo ""

# Test 5: Auto-sync Configuration
echo "🔄 Test 5: Auto-sync Enabled"
echo "----------------------------------------"
for app in "${HELM_APPS[@]}"; do
  app_file="${APPS_DIR}/${app}.yaml"

  if [ "$app" = "wealist-infrastructure" ]; then
    app_file="${APPS_DIR}/infrastructure.yaml"
  fi

  if [ ! -f "$app_file" ]; then
    continue
  fi

  # Check for automated sync
  if grep -q "automated:" "$app_file"; then
    print_result "${app} auto-sync enabled" 0
  else
    print_result "${app} auto-sync enabled" 1
  fi
done
echo ""

# Test 6: Namespace Configuration
echo "🏷️  Test 6: Namespace Configuration"
echo "----------------------------------------"
for app in "${HELM_APPS[@]}"; do
  app_file="${APPS_DIR}/${app}.yaml"

  if [ "$app" = "wealist-infrastructure" ]; then
    app_file="${APPS_DIR}/infrastructure.yaml"
  fi

  if [ ! -f "$app_file" ]; then
    continue
  fi

  # Check for destination namespace
  if grep -q "namespace: wealist-dev" "$app_file"; then
    print_result "${app} namespace configured" 0
  else
    print_result "${app} namespace configured" 1
  fi
done
echo ""

# Test 7: Project Assignment
echo "🎯 Test 7: Project Assignment"
echo "----------------------------------------"
for app in "${HELM_APPS[@]}"; do
  app_file="${APPS_DIR}/${app}.yaml"

  if [ "$app" = "wealist-infrastructure" ]; then
    app_file="${APPS_DIR}/infrastructure.yaml"
  fi

  if [ ! -f "$app_file" ]; then
    continue
  fi

  # Check for project
  if grep -q "project: wealist" "$app_file"; then
    print_result "${app} project assigned" 0
  else
    print_result "${app} project assigned" 1
  fi
done
echo ""

# Test 8: No Kustomize References
echo "🚫 Test 8: No Kustomize References"
echo "----------------------------------------"
for app in "${HELM_APPS[@]}"; do
  app_file="${APPS_DIR}/${app}.yaml"

  if [ "$app" = "wealist-infrastructure" ]; then
    app_file="${APPS_DIR}/infrastructure.yaml"
  fi

  if [ ! -f "$app_file" ]; then
    continue
  fi

  # Check for old Kustomize paths
  if grep -q "k8s/overlays" "$app_file"; then
    print_result "${app} no Kustomize refs" 1
  else
    print_result "${app} no Kustomize refs" 0
  fi
done
echo ""

# Summary
echo "========================================"
echo "📊 Validation Summary"
echo "========================================"
echo -e "Total Tests:  ${total_tests}"
echo -e "Passed:       ${GREEN}${passed_tests}${NC}"
echo -e "Failed:       ${RED}${failed_tests}${NC}"
echo ""

if [ $failed_tests -eq 0 ]; then
  echo -e "${GREEN}✓ All tests passed!${NC}"
  echo "ArgoCD Applications are ready for Helm! 🎉"
  echo ""
  echo "📋 Summary:"
  echo "  - Infrastructure: 1 Application"
  echo "  - Services: 8 Applications"
  echo "  - Total: 9 Helm-based Applications"
  echo ""
  echo "🚀 Next steps:"
  echo "  1. Commit changes: git add k8s/argocd/"
  echo "  2. Push to repository"
  echo "  3. Apply root-app: kubectl apply -f k8s/argocd/apps/root-app.yaml"
  echo "  4. Monitor sync: argocd app list"
  exit 0
else
  echo -e "${RED}✗ Some tests failed${NC}"
  echo "Please review and fix the failing Applications."
  exit 1
fi

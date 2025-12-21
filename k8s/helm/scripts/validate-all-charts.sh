#!/bin/bash
# =============================================================================
# Comprehensive Helm Chart Validation Script
# =============================================================================
# Validates all weAlist Helm charts for production readiness

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHARTS_DIR="$(cd "${SCRIPT_DIR}/../charts" && pwd)"

CHARTS=(
  "wealist-common"
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

SERVICE_CHARTS=(
  "auth-service"
  "user-service"
  "board-service"
  "chat-service"
  "noti-service"
  "storage-service"
  "video-service"
  "frontend"
)

echo "🔍 Helm Chart Validation Suite"
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

# Test 1: Chart.yaml existence
echo "📋 Test 1: Chart.yaml Files"
echo "----------------------------------------"
for chart in "${CHARTS[@]}"; do
  if [ -f "${CHARTS_DIR}/${chart}/Chart.yaml" ]; then
    print_result "${chart}/Chart.yaml exists" 0
  else
    print_result "${chart}/Chart.yaml exists" 1
  fi
done
echo ""

# Test 2: Helm Lint
echo "🔨 Test 2: Helm Lint (Production Values)"
echo "----------------------------------------"
for chart in "${CHARTS[@]}"; do
  if [ "$chart" = "wealist-common" ]; then
    # Library chart - just check structure
    if helm lint "${CHARTS_DIR}/${chart}" >/dev/null 2>&1; then
      print_result "${chart} lint" 0
    else
      print_result "${chart} lint" 1
    fi
  else
    # Application charts
    if helm lint "${CHARTS_DIR}/${chart}" >/dev/null 2>&1; then
      print_result "${chart} lint" 0
    else
      print_result "${chart} lint" 1
    fi
  fi
done
echo ""

# Test 3: Development Values Lint
echo "🔨 Test 3: Helm Lint (Development Values)"
echo "----------------------------------------"
for chart in "${SERVICE_CHARTS[@]}"; do
  if [ -f "${CHARTS_DIR}/${chart}/values-develop-registry-local.yaml" ]; then
    if helm lint "${CHARTS_DIR}/${chart}" -f "${CHARTS_DIR}/${chart}/values-develop-registry-local.yaml" >/dev/null 2>&1; then
      print_result "${chart} lint (dev)" 0
    else
      print_result "${chart} lint (dev)" 1
    fi
  else
    print_result "${chart} values-develop-registry-local.yaml exists" 1
  fi
done
echo ""

# Test 4: Template Rendering (Production)
echo "📝 Test 4: Template Rendering (Production Values)"
echo "----------------------------------------"
for chart in "${SERVICE_CHARTS[@]}"; do
  if helm template test "${CHARTS_DIR}/${chart}" >/dev/null 2>&1; then
    print_result "${chart} template render (prod)" 0
  else
    print_result "${chart} template render (prod)" 1
  fi
done
echo ""

# Test 5: Template Rendering (Development)
echo "📝 Test 5: Template Rendering (Development Values)"
echo "----------------------------------------"
for chart in "${SERVICE_CHARTS[@]}"; do
  if helm template test "${CHARTS_DIR}/${chart}" -f "${CHARTS_DIR}/${chart}/values-develop-registry-local.yaml" >/dev/null 2>&1; then
    print_result "${chart} template render (dev)" 0
  else
    print_result "${chart} template render (dev)" 1
  fi
done
echo ""

# Test 6: Required Values Check
echo "🔑 Test 6: Required Values Present"
echo "----------------------------------------"
for chart in "${SERVICE_CHARTS[@]}"; do
  values_file="${CHARTS_DIR}/${chart}/values.yaml"

  # Check for required fields (simplified)
  if grep -q "repository:" "$values_file" && \
     grep -q "port:" "$values_file" && \
     grep -q "targetPort:" "$values_file"; then
    print_result "${chart} required values" 0
  else
    print_result "${chart} required values" 1
  fi
done
echo ""

# Test 7: Security Context Present
echo "🛡️  Test 7: Security Contexts Configured"
echo "----------------------------------------"
for chart in "${SERVICE_CHARTS[@]}"; do
  values_file="${CHARTS_DIR}/${chart}/values.yaml"

  if grep -q "podSecurityContext" "$values_file" && grep -q "securityContext" "$values_file"; then
    print_result "${chart} security contexts" 0
  else
    print_result "${chart} security contexts" 1
  fi
done
echo ""

# Test 8: Production-Ready Features
echo "🎯 Test 8: Production Features"
echo "----------------------------------------"
for chart in "${SERVICE_CHARTS[@]}"; do
  values_file="${CHARTS_DIR}/${chart}/values.yaml"
  features=("autoscaling" "podDisruptionBudget" "resources")
  all_present=true

  for feature in "${features[@]}"; do
    if ! grep -q "$feature" "$values_file"; then
      all_present=false
      break
    fi
  done

  if $all_present; then
    print_result "${chart} production features" 0
  else
    print_result "${chart} production features" 1
  fi
done
echo ""

# Test 9: Dependency Check
echo "📦 Test 9: Dependencies Resolved"
echo "----------------------------------------"
for chart in "${SERVICE_CHARTS[@]}"; do
  if [ -f "${CHARTS_DIR}/${chart}/charts/wealist-common-1.0.0.tgz" ] || \
     [ -d "${CHARTS_DIR}/${chart}/charts/wealist-common" ]; then
    print_result "${chart} dependencies" 0
  else
    print_result "${chart} dependencies" 1
  fi
done
echo ""

# Test 10: Template File Consistency
echo "📄 Test 10: Template Files Present"
echo "----------------------------------------"
required_templates=("deployment.yaml" "service.yaml" "configmap.yaml")
for chart in "${SERVICE_CHARTS[@]}"; do
  all_present=true

  for template in "${required_templates[@]}"; do
    if [ ! -f "${CHARTS_DIR}/${chart}/templates/${template}" ]; then
      all_present=false
      break
    fi
  done

  if $all_present; then
    print_result "${chart} core templates" 0
  else
    print_result "${chart} core templates" 1
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
  echo "Charts are production-ready! 🎉"
  exit 0
else
  echo -e "${RED}✗ Some tests failed${NC}"
  echo "Please review and fix the failing charts."
  exit 1
fi

# =============================================================================
# weAlist Performance Testing Makefile
# =============================================================================
# k6를 사용한 성능 테스트 실행
#
# 사용법:
#   make k6-load              # Load Test 실행
#   make k6-stress            # Stress Test 실행
#   make k6-spike             # Spike Test 실행
#   make k6-all               # 전체 테스트 실행
#
# 환경변수:
#   BASE_URL                  # 테스트 대상 URL (기본: http://localhost:8080)
#   TEST_TOKEN                # 인증 토큰 (선택)
#   K6_PROMETHEUS_RW_URL      # Prometheus Remote Write URL (선택)
# =============================================================================

.PHONY: help k6-load k6-stress k6-spike k6-all k6-auth k6-board-crud k6-install check-k6

# =============================================================================
# Configuration
# =============================================================================
K6_SCRIPTS_DIR := k6/scripts
K6_SCENARIOS_DIR := k6/scripts/scenarios

# Default values
BASE_URL ?= http://localhost:8080
K6_PROMETHEUS_RW_URL ?= http://localhost:9090/api/v1/write
TEST_TOKEN ?=
WORKSPACE_ID ?= test-workspace

# k6 output options
K6_OUTPUT ?= --out experimental-prometheus-rw

# =============================================================================
# Help
# =============================================================================
help: ## Show this help
	@echo "weAlist Performance Testing"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Environment Variables:"
	@echo "  BASE_URL              Test target URL (default: http://localhost:8080)"
	@echo "  TEST_TOKEN            Auth token for authenticated endpoints"
	@echo "  K6_PROMETHEUS_RW_URL  Prometheus Remote Write endpoint"
	@echo "  WORKSPACE_ID          Test workspace ID (default: test-workspace)"

# =============================================================================
# Prerequisites
# =============================================================================
check-k6: ## Check if k6 is installed
	@command -v k6 >/dev/null 2>&1 || { echo "k6 is not installed. Install it from https://k6.io/docs/getting-started/installation/"; exit 1; }
	@echo "k6 version: $$(k6 version)"

k6-install: ## Install k6 (macOS)
	@echo "Installing k6..."
	@brew install k6 || { echo "Failed to install k6. Please install manually from https://k6.io"; exit 1; }
	@echo "k6 installed successfully"

# =============================================================================
# Load Testing
# =============================================================================
k6-load: check-k6 ## Run Load Test (20 VUs, 5 minutes)
	@echo "Starting Load Test..."
	@echo "Target: $(BASE_URL)"
	K6_PROMETHEUS_RW_SERVER_URL=$(K6_PROMETHEUS_RW_URL) \
	BASE_URL=$(BASE_URL) \
	TEST_TOKEN=$(TEST_TOKEN) \
	k6 run $(K6_OUTPUT) $(K6_SCRIPTS_DIR)/load-test.js

k6-load-local: check-k6 ## Run Load Test without Prometheus output
	@echo "Starting Load Test (local output only)..."
	BASE_URL=$(BASE_URL) \
	TEST_TOKEN=$(TEST_TOKEN) \
	k6 run $(K6_SCRIPTS_DIR)/load-test.js

# =============================================================================
# Stress Testing
# =============================================================================
k6-stress: check-k6 ## Run Stress Test (50→300 VUs, ~30 minutes)
	@echo "Starting Stress Test..."
	@echo "Target: $(BASE_URL)"
	@echo "WARNING: This test will put significant load on the system"
	K6_PROMETHEUS_RW_SERVER_URL=$(K6_PROMETHEUS_RW_URL) \
	BASE_URL=$(BASE_URL) \
	TEST_TOKEN=$(TEST_TOKEN) \
	k6 run $(K6_OUTPUT) $(K6_SCRIPTS_DIR)/stress-test.js

k6-stress-local: check-k6 ## Run Stress Test without Prometheus output
	BASE_URL=$(BASE_URL) \
	TEST_TOKEN=$(TEST_TOKEN) \
	k6 run $(K6_SCRIPTS_DIR)/stress-test.js

# =============================================================================
# Spike Testing
# =============================================================================
k6-spike: check-k6 ## Run Spike Test (20→500 VUs burst, ~10 minutes)
	@echo "Starting Spike Test..."
	@echo "Target: $(BASE_URL)"
	@echo "WARNING: This test will simulate sudden traffic bursts"
	K6_PROMETHEUS_RW_SERVER_URL=$(K6_PROMETHEUS_RW_URL) \
	BASE_URL=$(BASE_URL) \
	k6 run $(K6_OUTPUT) $(K6_SCRIPTS_DIR)/spike-test.js

k6-spike-local: check-k6 ## Run Spike Test without Prometheus output
	BASE_URL=$(BASE_URL) \
	k6 run $(K6_SCRIPTS_DIR)/spike-test.js

# =============================================================================
# Scenario Tests
# =============================================================================
k6-auth: check-k6 ## Run Authentication Flow scenario
	@echo "Starting Auth Flow Scenario..."
	BASE_URL=$(BASE_URL) \
	k6 run $(K6_SCENARIOS_DIR)/auth-flow.js

k6-board-crud: check-k6 ## Run Board CRUD scenario
	@echo "Starting Board CRUD Scenario..."
	BASE_URL=$(BASE_URL) \
	TEST_TOKEN=$(TEST_TOKEN) \
	WORKSPACE_ID=$(WORKSPACE_ID) \
	k6 run $(K6_SCENARIOS_DIR)/board-crud.js

# =============================================================================
# Combined Tests
# =============================================================================
k6-all: check-k6 ## Run all performance tests sequentially
	@echo "Running all performance tests..."
	@echo ""
	@echo "=== Load Test ==="
	@$(MAKE) k6-load-local
	@echo ""
	@echo "=== Stress Test ==="
	@$(MAKE) k6-stress-local
	@echo ""
	@echo "=== Spike Test ==="
	@$(MAKE) k6-spike-local
	@echo ""
	@echo "All tests completed!"

k6-scenarios: check-k6 ## Run all scenario tests
	@echo "Running scenario tests..."
	@$(MAKE) k6-auth
	@$(MAKE) k6-board-crud

# =============================================================================
# Cleanup
# =============================================================================
clean: ## Clean up generated files
	@rm -f *.json
	@rm -f k6/scripts/*.json
	@echo "Cleaned up generated files"

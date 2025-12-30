# =============================================================================
# k6 Performance Testing Commands
# =============================================================================
# k6를 사용한 성능 테스트 실행
#
# 로컬 실행:
#   make k6-load              # Load Test (20 VUs, 5분)
#   make k6-stress            # Stress Test (50→300 VUs)
#   make k6-spike             # Spike Test (20→500 VUs)
#
# K8s 클러스터에서 실행:
#   make k6-job-load          # Load Test Job 생성
#   make k6-job-stress        # Stress Test Job 생성
#   make k6-job-spike         # Spike Test Job 생성
# =============================================================================

##@ Performance Testing (k6)

# =============================================================================
# Configuration
# =============================================================================
K6_SCRIPTS_DIR := tests/performance/k6/scripts
K6_BASE_URL ?= http://localhost:8080
K6_PROMETHEUS_RW_URL ?= http://localhost:9090/api/v1/write
K6_TEST_TOKEN ?=

# =============================================================================
# Local Execution (requires k6 installed locally)
# =============================================================================
.PHONY: k6-check k6-load k6-stress k6-spike k6-all

k6-check: ## Check if k6 is installed
	@command -v k6 >/dev/null 2>&1 || { echo "k6 is not installed. Install from https://k6.io"; exit 1; }
	@echo "k6 version: $$(k6 version)"

k6-load: k6-check ## Run Load Test locally (20 VUs, 5 minutes)
	@echo "Starting Load Test..."
	@echo "Target: $(K6_BASE_URL)"
	BASE_URL=$(K6_BASE_URL) \
	TEST_TOKEN=$(K6_TEST_TOKEN) \
	k6 run $(K6_SCRIPTS_DIR)/load-test.js

k6-load-prometheus: k6-check ## Run Load Test with Prometheus output
	@echo "Starting Load Test with Prometheus output..."
	K6_PROMETHEUS_RW_SERVER_URL=$(K6_PROMETHEUS_RW_URL) \
	BASE_URL=$(K6_BASE_URL) \
	TEST_TOKEN=$(K6_TEST_TOKEN) \
	k6 run --out experimental-prometheus-rw $(K6_SCRIPTS_DIR)/load-test.js

k6-stress: k6-check ## Run Stress Test locally (50→300 VUs, ~30 minutes)
	@echo "Starting Stress Test..."
	@echo "WARNING: This will put significant load on the system"
	BASE_URL=$(K6_BASE_URL) \
	TEST_TOKEN=$(K6_TEST_TOKEN) \
	k6 run $(K6_SCRIPTS_DIR)/stress-test.js

k6-spike: k6-check ## Run Spike Test locally (20→500 VUs burst)
	@echo "Starting Spike Test..."
	@echo "WARNING: This will simulate sudden traffic bursts"
	BASE_URL=$(K6_BASE_URL) \
	k6 run $(K6_SCRIPTS_DIR)/spike-test.js

k6-all: k6-check ## Run all performance tests sequentially
	@echo "Running all performance tests..."
	@$(MAKE) k6-load
	@$(MAKE) k6-stress
	@$(MAKE) k6-spike
	@echo "All tests completed!"

# =============================================================================
# Scenario Tests
# =============================================================================
.PHONY: k6-auth k6-board-crud k6-scenarios

k6-auth: k6-check ## Run Authentication Flow scenario
	@echo "Starting Auth Flow Scenario..."
	BASE_URL=$(K6_BASE_URL) \
	k6 run $(K6_SCRIPTS_DIR)/scenarios/auth-flow.js

k6-board-crud: k6-check ## Run Board CRUD scenario
	@echo "Starting Board CRUD Scenario..."
	BASE_URL=$(K6_BASE_URL) \
	TEST_TOKEN=$(K6_TEST_TOKEN) \
	k6 run $(K6_SCRIPTS_DIR)/scenarios/board-crud.js

k6-scenarios: k6-check ## Run all scenario tests
	@$(MAKE) k6-auth
	@$(MAKE) k6-board-crud

# =============================================================================
# Kubernetes Job Execution
# =============================================================================
.PHONY: k6-job-load k6-job-stress k6-job-spike k6-job-status k6-job-logs k6-job-clean

k6-job-load: ## Create k6 Load Test Job in K8s
	@echo "Creating k6 Load Test Job in namespace $(K8S_NAMESPACE)..."
	kubectl delete job k6-load-test -n $(K8S_NAMESPACE) --ignore-not-found
	kubectl create job k6-load-test-$$(date +%Y%m%d-%H%M%S) \
		--from=job/k6-load-test \
		-n $(K8S_NAMESPACE) 2>/dev/null || \
	kubectl apply -f - <<EOF
	apiVersion: batch/v1
	kind: Job
	metadata:
	  name: k6-load-test-$$(date +%Y%m%d-%H%M%S)
	  namespace: $(K8S_NAMESPACE)
	spec:
	  template:
	    spec:
	      containers:
	      - name: k6
	        image: grafana/k6:0.49.0
	        command: ["k6", "run", "--out", "experimental-prometheus-rw", "/scripts/load-test.js"]
	        env:
	        - name: K6_PROMETHEUS_RW_SERVER_URL
	          value: "http://prometheus:9090/api/v1/write"
	        volumeMounts:
	        - name: scripts
	          mountPath: /scripts
	      restartPolicy: Never
	      volumes:
	      - name: scripts
	        configMap:
	          name: k6-scripts
	  backoffLimit: 0
	EOF
	@echo "Job created. Check status with: make k6-job-status"

k6-job-stress: ## Create k6 Stress Test Job in K8s
	@echo "Creating k6 Stress Test Job..."
	kubectl delete job k6-stress-test -n $(K8S_NAMESPACE) --ignore-not-found
	kubectl create job k6-stress-test-$$(date +%Y%m%d-%H%M%S) \
		--from=job/k6-stress-test \
		-n $(K8S_NAMESPACE) || echo "Using Helm-deployed job template"

k6-job-spike: ## Create k6 Spike Test Job in K8s
	@echo "Creating k6 Spike Test Job..."
	kubectl delete job k6-spike-test -n $(K8S_NAMESPACE) --ignore-not-found
	kubectl create job k6-spike-test-$$(date +%Y%m%d-%H%M%S) \
		--from=job/k6-spike-test \
		-n $(K8S_NAMESPACE) || echo "Using Helm-deployed job template"

k6-job-status: ## Show k6 Job status
	@echo "k6 Jobs in namespace $(K8S_NAMESPACE):"
	kubectl get jobs -n $(K8S_NAMESPACE) -l app=k6 --sort-by=.metadata.creationTimestamp

k6-job-logs: ## Show k6 Job logs
	@echo "Showing latest k6 Job logs..."
	@POD=$$(kubectl get pods -n $(K8S_NAMESPACE) -l app=k6 --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null); \
	if [ -n "$$POD" ]; then \
		kubectl logs $$POD -n $(K8S_NAMESPACE); \
	else \
		echo "No k6 pods found"; \
	fi

k6-job-clean: ## Clean up completed k6 Jobs
	@echo "Cleaning up completed k6 Jobs..."
	kubectl delete jobs -n $(K8S_NAMESPACE) -l app=k6 --field-selector status.successful=1
	@echo "Done"

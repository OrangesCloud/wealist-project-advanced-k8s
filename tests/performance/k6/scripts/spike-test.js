/**
 * k6 Spike Test - weAlist
 *
 * 갑작스러운 트래픽 급증 시 시스템 반응 측정
 * - 목표: 급격한 부하 증가 시 시스템 복원력 확인
 * - VUs: 20 → 500 (급증) → 20 → 500 (반복)
 * - 기간: 약 10분
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';

// =============================================================================
// Custom Metrics
// =============================================================================
const errorRate = new Rate('errors');
const spikeLatency = new Trend('spike_latency', true);
const recoveryLatency = new Trend('recovery_latency', true);
const spikeRequests = new Counter('spike_requests');
const recoveryRequests = new Counter('recovery_requests');
const spikePhase = new Gauge('spike_phase');  // 0=normal, 1=spike

// =============================================================================
// Test Configuration
// =============================================================================
export const options = {
  scenarios: {
    spike_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        // Warm up
        { duration: '30s', target: 20 },
        { duration: '1m', target: 20 },

        // First Spike!
        { duration: '10s', target: 500 },
        { duration: '1m', target: 500 },

        // Recovery
        { duration: '10s', target: 20 },
        { duration: '1m', target: 20 },

        // Second Spike!
        { duration: '10s', target: 500 },
        { duration: '1m', target: 500 },

        // Final Recovery
        { duration: '10s', target: 20 },
        { duration: '2m', target: 20 },

        // Cool down
        { duration: '30s', target: 0 },
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<3000'],  // Very relaxed during spike
    http_req_failed: ['rate<0.2'],      // Allow up to 20% errors during spike
    errors: ['rate<0.25'],
    spike_latency: ['p(95)<5000'],      // Spike phase can be slow
    recovery_latency: ['p(95)<1000'],   // Recovery should be faster
  },
};

// =============================================================================
// Environment Configuration
// =============================================================================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// =============================================================================
// Spike Detection Helper
// =============================================================================
function isSpikePeriod() {
  // Determine if we're in a spike period based on VU count
  // Spike threshold: > 100 VUs
  return __VU > 50;
}

// =============================================================================
// Spike Test Scenarios
// =============================================================================
export default function () {
  const inSpike = isSpikePeriod();
  spikePhase.add(inSpike ? 1 : 0);

  // Primary endpoint - health check (lightest load)
  const healthRes = http.get(`${BASE_URL}/svc/board/health/live`, {
    tags: {
      name: 'health-spike',
      phase: inSpike ? 'spike' : 'normal',
    },
  });

  const success = healthRes.status === 200;
  errorRate.add(!success);

  if (inSpike) {
    spikeLatency.add(healthRes.timings.duration);
    spikeRequests.add(1);
  } else {
    recoveryLatency.add(healthRes.timings.duration);
    recoveryRequests.add(1);
  }

  check(healthRes, {
    'health check passed': (r) => r.status === 200,
    'response time acceptable': (r) => r.timings.duration < (inSpike ? 5000 : 1000),
  });

  // API endpoints during spike
  const apiRes = http.get(`${BASE_URL}/svc/board/api/boards`, {
    tags: {
      name: 'boards-spike',
      phase: inSpike ? 'spike' : 'normal',
    },
  });

  const apiSuccess = apiRes.status >= 200 && apiRes.status < 500;
  errorRate.add(!apiSuccess);

  if (inSpike) {
    spikeLatency.add(apiRes.timings.duration);
    spikeRequests.add(1);
  } else {
    recoveryLatency.add(apiRes.timings.duration);
    recoveryRequests.add(1);
  }

  check(apiRes, {
    'API responds during spike': (r) => r.status < 500,
  });

  // Very short sleep during spike to maximize load
  sleep(inSpike ? 0.1 : 0.5);
}

// =============================================================================
// Setup & Teardown
// =============================================================================
export function setup() {
  console.log('='.repeat(60));
  console.log('SPIKE TEST STARTING');
  console.log(`Target: ${BASE_URL}`);
  console.log('Pattern: 20 VUs → 500 VUs (spike) → 20 VUs (recover)');
  console.log('Testing system resilience to sudden traffic bursts');
  console.log('='.repeat(60));

  // Baseline measurement
  const baseline = http.get(`${BASE_URL}/svc/board/health/live`);

  return {
    startTime: new Date().toISOString(),
    baselineLatency: baseline.timings.duration,
    baselineStatus: baseline.status,
  };
}

export function teardown(data) {
  // Post-spike health check
  const recovery = http.get(`${BASE_URL}/svc/board/health/live`);

  console.log('='.repeat(60));
  console.log('SPIKE TEST COMPLETED');
  console.log(`Started: ${data.startTime}`);
  console.log(`Baseline Latency: ${data.baselineLatency}ms`);
  console.log(`Post-Spike Latency: ${recovery.timings.duration}ms`);
  console.log(`Recovery Status: ${recovery.status === 200 ? 'HEALTHY' : 'DEGRADED'}`);
  console.log('='.repeat(60));
}

// =============================================================================
// Custom Summary
// =============================================================================
export function handleSummary(data) {
  const spikeP95 = data.metrics.spike_latency?.values['p(95)'] || 0;
  const recoveryP95 = data.metrics.recovery_latency?.values['p(95)'] || 0;

  const summary = {
    testType: 'spike',
    timestamp: new Date().toISOString(),
    totalRequests: data.metrics.http_reqs?.values?.count || 0,
    spikeRequests: data.metrics.spike_requests?.values?.count || 0,
    recoveryRequests: data.metrics.recovery_requests?.values?.count || 0,
    overallErrorRate: data.metrics.http_req_failed?.values?.rate || 0,
    spike: {
      p95Latency: spikeP95,
      p99Latency: data.metrics.spike_latency?.values['p(99)'] || 0,
    },
    recovery: {
      p95Latency: recoveryP95,
      p99Latency: data.metrics.recovery_latency?.values['p(99)'] || 0,
    },
    resilience: {
      latencyDegradation: spikeP95 > 0 ? (spikeP95 / recoveryP95).toFixed(2) + 'x' : 'N/A',
      recovered: recoveryP95 < 1000 ? 'YES' : 'DEGRADED',
    },
  };

  return {
    'stdout': JSON.stringify(summary, null, 2) + '\n',
    'spike-test-summary.json': JSON.stringify(summary, null, 2),
  };
}

/**
 * k6 Stress Test - weAlist
 *
 * 시스템의 한계점(breaking point) 확인
 * - 목표: 부하를 점진적으로 증가시켜 시스템 한계 파악
 * - VUs: 0 → 50 → 100 → 200 → 300 → 0
 * - 기간: 약 30분
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';

// =============================================================================
// Custom Metrics
// =============================================================================
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time', true);
const requestsPerSecond = new Counter('requests_per_second');
const currentVUs = new Gauge('current_vus');
const breakingPointHit = new Counter('breaking_point_hit');

// =============================================================================
// Test Configuration
// =============================================================================
export const options = {
  scenarios: {
    stress_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        // Stage 1: Warm up
        { duration: '2m', target: 50 },
        { duration: '3m', target: 50 },

        // Stage 2: Normal load
        { duration: '2m', target: 100 },
        { duration: '3m', target: 100 },

        // Stage 3: High load
        { duration: '2m', target: 200 },
        { duration: '3m', target: 200 },

        // Stage 4: Breaking point search
        { duration: '2m', target: 300 },
        { duration: '5m', target: 300 },

        // Stage 5: Recovery
        { duration: '2m', target: 100 },
        { duration: '2m', target: 0 },
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000'],  // Relaxed for stress test
    http_req_failed: ['rate<0.1'],      // Allow up to 10% errors
    errors: ['rate<0.15'],              // Allow up to 15% errors
  },
};

// =============================================================================
// Environment Configuration
// =============================================================================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TEST_TOKEN = __ENV.TEST_TOKEN || '';

const headers = {
  'Content-Type': 'application/json',
};

if (TEST_TOKEN) {
  headers['Authorization'] = `Bearer ${TEST_TOKEN}`;
}

// =============================================================================
// Stress Test Scenarios
// =============================================================================
export default function () {
  currentVUs.add(__VU);

  // Batch requests to multiple services
  const responses = http.batch([
    ['GET', `${BASE_URL}/svc/board/health/live`, null, { tags: { name: 'board-health' } }],
    ['GET', `${BASE_URL}/svc/user/health/live`, null, { tags: { name: 'user-health' } }],
    ['GET', `${BASE_URL}/svc/chat/health/live`, null, { tags: { name: 'chat-health' } }],
    ['GET', `${BASE_URL}/svc/noti/health/live`, null, { tags: { name: 'noti-health' } }],
  ]);

  let hasError = false;

  responses.forEach((res, i) => {
    const success = res.status >= 200 && res.status < 400;
    if (!success) {
      hasError = true;
      breakingPointHit.add(1);
    }

    responseTime.add(res.timings.duration);
    requestsPerSecond.add(1);

    check(res, {
      [`batch request ${i + 1} successful`]: (r) => r.status >= 200 && r.status < 400,
      [`batch request ${i + 1} response time < 2s`]: (r) => r.timings.duration < 2000,
    });
  });

  errorRate.add(hasError);

  // Board API under stress
  group('Board API Stress', function () {
    const boardRes = http.get(`${BASE_URL}/svc/board/api/boards`, {
      headers: headers,
      tags: { name: 'board-list-stress' },
    });

    responseTime.add(boardRes.timings.duration);
    requestsPerSecond.add(1);

    const success = boardRes.status >= 200 && boardRes.status < 400;
    if (!success && boardRes.status !== 401) {
      breakingPointHit.add(1);
    }

    check(boardRes, {
      'board list under stress': (r) => r.status >= 200 && r.status < 500,
    });
  });

  // Shorter sleep during stress to increase load
  sleep(0.3);
}

// =============================================================================
// Setup & Teardown
// =============================================================================
export function setup() {
  console.log('='.repeat(60));
  console.log('STRESS TEST STARTING');
  console.log(`Target: ${BASE_URL}`);
  console.log('Stages: 50 → 100 → 200 → 300 VUs');
  console.log('='.repeat(60));

  // Pre-flight check
  const res = http.get(`${BASE_URL}/svc/board/health/live`);
  if (res.status !== 200) {
    console.warn('WARNING: Services may not be fully healthy');
  }

  return {
    startTime: new Date().toISOString(),
    baselineLatency: res.timings.duration,
  };
}

export function teardown(data) {
  console.log('='.repeat(60));
  console.log('STRESS TEST COMPLETED');
  console.log(`Started: ${data.startTime}`);
  console.log(`Baseline Latency: ${data.baselineLatency}ms`);
  console.log('='.repeat(60));
}

// =============================================================================
// Custom Summary
// =============================================================================
export function handleSummary(data) {
  const summary = {
    testType: 'stress',
    timestamp: new Date().toISOString(),
    totalRequests: data.metrics.http_reqs?.values?.count || 0,
    errorRate: data.metrics.http_req_failed?.values?.rate || 0,
    p95Latency: data.metrics.http_req_duration?.values['p(95)'] || 0,
    p99Latency: data.metrics.http_req_duration?.values['p(99)'] || 0,
    maxVUs: data.metrics.vus_max?.values?.max || 0,
    breakingPointHits: data.metrics.breaking_point_hit?.values?.count || 0,
  };

  return {
    'stdout': JSON.stringify(summary, null, 2) + '\n',
    'stress-test-summary.json': JSON.stringify(summary, null, 2),
  };
}

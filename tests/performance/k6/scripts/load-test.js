/**
 * k6 Load Test - weAlist
 *
 * 정상 부하 상황에서의 시스템 성능 측정
 * - 목표: 일반적인 트래픽 패턴에서 안정적인 응답 시간 확인
 * - VUs: 1 → 20 → 20 유지 → 0
 * - 기간: 5분
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// =============================================================================
// Custom Metrics
// =============================================================================
const errorRate = new Rate('errors');
const boardLatency = new Trend('board_latency', true);
const userLatency = new Trend('user_latency', true);
const healthLatency = new Trend('health_latency', true);
const requestCount = new Counter('request_count');

// =============================================================================
// Test Configuration
// =============================================================================
export const options = {
  scenarios: {
    load_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 20 },  // Ramp up to 20 VUs
        { duration: '3m', target: 20 },  // Stay at 20 VUs
        { duration: '1m', target: 0 },   // Ramp down
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.01'],
    errors: ['rate<0.01'],
    board_latency: ['p(95)<400'],
    user_latency: ['p(95)<400'],
  },
  // Prometheus Remote Write output
  // Run with: k6 run --out experimental-prometheus-rw load-test.js
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
// Helper Functions
// =============================================================================
function handleResponse(res, metricTrend, name) {
  const success = res.status >= 200 && res.status < 400;
  errorRate.add(!success);
  requestCount.add(1);

  if (metricTrend) {
    metricTrend.add(res.timings.duration);
  }

  check(res, {
    [`${name} status is successful`]: (r) => r.status >= 200 && r.status < 400,
    [`${name} response time < 1s`]: (r) => r.timings.duration < 1000,
  });

  return success;
}

// =============================================================================
// Test Scenarios
// =============================================================================
export default function () {
  // Health Check (가장 가벼운 요청)
  group('Health Checks', function () {
    const services = ['board', 'user', 'chat', 'noti', 'storage', 'video'];

    for (const svc of services) {
      const res = http.get(`${BASE_URL}/svc/${svc}/health/live`, {
        tags: { name: `${svc}-health` },
      });
      handleResponse(res, healthLatency, `${svc} health`);
    }
  });

  sleep(0.5);

  // Board Service API
  group('Board Service', function () {
    // List boards
    const listRes = http.get(`${BASE_URL}/svc/board/api/boards`, {
      headers: headers,
      tags: { name: 'board-list' },
    });
    handleResponse(listRes, boardLatency, 'board list');
  });

  sleep(0.5);

  // User Service API
  group('User Service', function () {
    // Get user profile (requires auth)
    const profileRes = http.get(`${BASE_URL}/svc/user/api/users/me`, {
      headers: headers,
      tags: { name: 'user-profile' },
    });
    // 401 is expected without valid token
    const success = profileRes.status === 200 || profileRes.status === 401;
    errorRate.add(!success);
    userLatency.add(profileRes.timings.duration);
  });

  sleep(1);
}

// =============================================================================
// Setup & Teardown
// =============================================================================
export function setup() {
  console.log(`Starting Load Test against ${BASE_URL}`);
  console.log(`Test Token: ${TEST_TOKEN ? 'Provided' : 'Not provided'}`);

  // Verify services are up
  const healthRes = http.get(`${BASE_URL}/svc/board/health/live`);
  if (healthRes.status !== 200) {
    console.error('Board service is not healthy!');
  }

  return { startTime: new Date().toISOString() };
}

export function teardown(data) {
  console.log(`Load Test completed. Started at: ${data.startTime}`);
}

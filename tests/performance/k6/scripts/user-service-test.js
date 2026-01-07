/**
 * k6 User Service Test - weAlist
 *
 * user-service 전용 트래픽 생성 (카나리 배포 관측용)
 * - VUs: 1 → 10 → 10 유지 → 0
 * - 기간: 3분
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// =============================================================================
// Custom Metrics
// =============================================================================
const errorRate = new Rate('errors');
const userLatency = new Trend('user_latency', true);
const healthLatency = new Trend('health_latency', true);
const requestCount = new Counter('request_count');

// =============================================================================
// Test Configuration
// =============================================================================
export const options = {
  scenarios: {
    user_service_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 10 },  // Ramp up to 10 VUs
        { duration: '2m', target: 10 },   // Stay at 10 VUs
        { duration: '30s', target: 0 },   // Ramp down
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.05'],
    errors: ['rate<0.05'],
    user_latency: ['p(95)<400'],
  },
};

// =============================================================================
// Environment Configuration
// =============================================================================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TEST_TOKEN = __ENV.TEST_TOKEN || '';
const SVC_PREFIX = __ENV.SVC_PREFIX || '/svc';

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
    [`${name} response time < 500ms`]: (r) => r.timings.duration < 500,
  });

  return success;
}

// =============================================================================
// Test Scenarios - User Service Only
// =============================================================================
export default function () {
  // Health Check
  group('User Service Health', function () {
    const liveRes = http.get(`${BASE_URL}${SVC_PREFIX}/user/health/live`, {
      tags: { name: 'user-health-live' },
    });
    handleResponse(liveRes, healthLatency, 'user health live');

    const readyRes = http.get(`${BASE_URL}${SVC_PREFIX}/user/health/ready`, {
      tags: { name: 'user-health-ready' },
    });
    handleResponse(readyRes, healthLatency, 'user health ready');
  });

  sleep(0.3);

  // User Profile API
  group('User Profile', function () {
    const profileRes = http.get(`${BASE_URL}${SVC_PREFIX}/user/api/users/me`, {
      headers: headers,
      tags: { name: 'user-profile-me' },
    });
    // 401 is expected without valid token
    const success = profileRes.status === 200 || profileRes.status === 401;
    errorRate.add(!success);
    userLatency.add(profileRes.timings.duration);
  });

  sleep(0.3);

  // Workspaces API
  group('User Workspaces', function () {
    const workspacesRes = http.get(`${BASE_URL}${SVC_PREFIX}/user/api/workspaces`, {
      headers: headers,
      tags: { name: 'user-workspaces' },
    });
    // 401 is expected without valid token
    const success = workspacesRes.status === 200 || workspacesRes.status === 401;
    errorRate.add(!success);
    userLatency.add(workspacesRes.timings.duration);
  });

  sleep(0.5);
}

// =============================================================================
// Setup & Teardown
// =============================================================================
export function setup() {
  console.log(`Starting User Service Test against ${BASE_URL}`);
  console.log(`Service prefix: ${SVC_PREFIX}`);
  console.log(`Test Token: ${TEST_TOKEN ? 'Provided' : 'Not provided'}`);

  // Verify user-service is up
  const healthRes = http.get(`${BASE_URL}${SVC_PREFIX}/user/health/live`);
  if (healthRes.status !== 200) {
    console.error('User service is not healthy!');
  }

  return { startTime: new Date().toISOString() };
}

export function teardown(data) {
  console.log(`User Service Test completed. Started at: ${data.startTime}`);
}

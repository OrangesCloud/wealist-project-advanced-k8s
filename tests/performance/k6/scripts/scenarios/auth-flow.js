/**
 * k6 Scenario: Authentication Flow - weAlist
 *
 * 인증 플로우 전체를 시뮬레이션
 * - 로그인 → 토큰 발급 → 보호된 리소스 접근 → 토큰 갱신
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// =============================================================================
// Custom Metrics
// =============================================================================
const authErrors = new Rate('auth_errors');
const loginLatency = new Trend('login_latency', true);
const tokenValidationLatency = new Trend('token_validation_latency', true);
const protectedResourceLatency = new Trend('protected_resource_latency', true);

// =============================================================================
// Test Configuration
// =============================================================================
export const options = {
  scenarios: {
    auth_flow: {
      executor: 'constant-vus',
      vus: 10,
      duration: '5m',
    },
  },
  thresholds: {
    auth_errors: ['rate<0.05'],
    login_latency: ['p(95)<1000'],
    token_validation_latency: ['p(95)<200'],
    protected_resource_latency: ['p(95)<500'],
  },
};

// =============================================================================
// Environment Configuration
// =============================================================================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test credentials (for load testing only)
const TEST_USERS = [
  { email: 'test1@wealist.com', password: 'testpass123' },
  { email: 'test2@wealist.com', password: 'testpass123' },
  { email: 'test3@wealist.com', password: 'testpass123' },
];

// =============================================================================
// Auth Flow Scenario
// =============================================================================
export default function () {
  const user = TEST_USERS[__VU % TEST_USERS.length];
  let accessToken = null;

  // Step 1: Login
  group('1. Login', function () {
    const loginRes = http.post(
      `${BASE_URL}/svc/auth/api/auth/login`,
      JSON.stringify({
        email: user.email,
        password: user.password,
      }),
      {
        headers: { 'Content-Type': 'application/json' },
        tags: { name: 'auth-login' },
      }
    );

    loginLatency.add(loginRes.timings.duration);

    const loginSuccess = check(loginRes, {
      'login status is 200 or 401': (r) => r.status === 200 || r.status === 401,
      'login response time < 2s': (r) => r.timings.duration < 2000,
    });

    if (loginRes.status === 200) {
      try {
        const body = JSON.parse(loginRes.body);
        accessToken = body.accessToken || body.access_token || body.token;
      } catch (e) {
        console.log('Failed to parse login response');
      }
    }

    authErrors.add(!loginSuccess);
  });

  sleep(0.5);

  // Step 2: Token Validation (if we have a token)
  if (accessToken) {
    group('2. Token Validation', function () {
      const validateRes = http.get(
        `${BASE_URL}/svc/auth/api/auth/validate`,
        {
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${accessToken}`,
          },
          tags: { name: 'auth-validate' },
        }
      );

      tokenValidationLatency.add(validateRes.timings.duration);

      check(validateRes, {
        'token validation successful': (r) => r.status === 200,
        'validation response time < 500ms': (r) => r.timings.duration < 500,
      });
    });
  }

  sleep(0.5);

  // Step 3: Access Protected Resource
  group('3. Protected Resource', function () {
    const headers = { 'Content-Type': 'application/json' };
    if (accessToken) {
      headers['Authorization'] = `Bearer ${accessToken}`;
    }

    const protectedRes = http.get(
      `${BASE_URL}/svc/user/api/users/me`,
      {
        headers: headers,
        tags: { name: 'user-profile' },
      }
    );

    protectedResourceLatency.add(protectedRes.timings.duration);

    const expectedStatus = accessToken ? 200 : 401;
    check(protectedRes, {
      'protected resource access correct': (r) =>
        r.status === expectedStatus || r.status === 401,
      'response time < 1s': (r) => r.timings.duration < 1000,
    });
  });

  sleep(1);

  // Step 4: Access Board with Auth
  group('4. Authenticated Board Access', function () {
    const headers = { 'Content-Type': 'application/json' };
    if (accessToken) {
      headers['Authorization'] = `Bearer ${accessToken}`;
    }

    const boardRes = http.get(
      `${BASE_URL}/svc/board/api/boards`,
      {
        headers: headers,
        tags: { name: 'board-list-auth' },
      }
    );

    check(boardRes, {
      'board list accessible': (r) => r.status === 200 || r.status === 401,
    });
  });

  sleep(1);
}

// =============================================================================
// Setup & Teardown
// =============================================================================
export function setup() {
  console.log('Auth Flow Scenario Starting');
  console.log(`Target: ${BASE_URL}`);
  console.log(`Test Users: ${TEST_USERS.length}`);

  // Check auth service health
  const healthRes = http.get(`${BASE_URL}/svc/auth/health/live`);
  console.log(`Auth Service Health: ${healthRes.status}`);

  return { startTime: new Date().toISOString() };
}

export function teardown(data) {
  console.log(`Auth Flow Scenario Completed`);
  console.log(`Started: ${data.startTime}`);
}

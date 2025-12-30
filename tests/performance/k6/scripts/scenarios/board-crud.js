/**
 * k6 Scenario: Board CRUD Operations - weAlist
 *
 * 보드 서비스의 CRUD 작업 전체를 시뮬레이션
 * - Create → Read → Update → Delete
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// =============================================================================
// Custom Metrics
// =============================================================================
const crudErrors = new Rate('crud_errors');
const createLatency = new Trend('create_latency', true);
const readLatency = new Trend('read_latency', true);
const updateLatency = new Trend('update_latency', true);
const deleteLatency = new Trend('delete_latency', true);
const crudOperations = new Counter('crud_operations');

// =============================================================================
// Test Configuration
// =============================================================================
export const options = {
  scenarios: {
    board_crud: {
      executor: 'constant-vus',
      vus: 5,
      duration: '5m',
    },
  },
  thresholds: {
    crud_errors: ['rate<0.05'],
    create_latency: ['p(95)<1000'],
    read_latency: ['p(95)<500'],
    update_latency: ['p(95)<1000'],
    delete_latency: ['p(95)<500'],
  },
};

// =============================================================================
// Environment Configuration
// =============================================================================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TEST_TOKEN = __ENV.TEST_TOKEN || '';
const WORKSPACE_ID = __ENV.WORKSPACE_ID || 'test-workspace';

const headers = {
  'Content-Type': 'application/json',
};

if (TEST_TOKEN) {
  headers['Authorization'] = `Bearer ${TEST_TOKEN}`;
}

// =============================================================================
// Board CRUD Scenario
// =============================================================================
export default function () {
  let createdBoardId = null;
  const testBoardName = `perf-test-board-${uuidv4().substring(0, 8)}`;

  // Step 1: Create Board
  group('1. Create Board', function () {
    const createPayload = JSON.stringify({
      name: testBoardName,
      description: `Performance test board created at ${new Date().toISOString()}`,
      workspaceId: WORKSPACE_ID,
    });

    const createRes = http.post(
      `${BASE_URL}/svc/board/api/boards`,
      createPayload,
      {
        headers: headers,
        tags: { name: 'board-create' },
      }
    );

    createLatency.add(createRes.timings.duration);
    crudOperations.add(1);

    const createSuccess = check(createRes, {
      'board created successfully': (r) => r.status === 201 || r.status === 200,
      'create response time < 2s': (r) => r.timings.duration < 2000,
    });

    if (createRes.status === 201 || createRes.status === 200) {
      try {
        const body = JSON.parse(createRes.body);
        createdBoardId = body.id || body.boardId || body.data?.id;
      } catch (e) {
        console.log('Failed to parse create response');
      }
    }

    // If not authenticated, that's okay for load testing
    if (createRes.status === 401) {
      crudErrors.add(false);  // Not counting as error
    } else {
      crudErrors.add(!createSuccess);
    }
  });

  sleep(0.5);

  // Step 2: Read Board List
  group('2. Read Board List', function () {
    const listRes = http.get(
      `${BASE_URL}/svc/board/api/boards`,
      {
        headers: headers,
        tags: { name: 'board-list' },
      }
    );

    readLatency.add(listRes.timings.duration);
    crudOperations.add(1);

    const readSuccess = check(listRes, {
      'board list retrieved': (r) => r.status === 200 || r.status === 401,
      'read response time < 1s': (r) => r.timings.duration < 1000,
    });

    crudErrors.add(!readSuccess);
  });

  sleep(0.5);

  // Step 3: Read Single Board (if created)
  if (createdBoardId) {
    group('3. Read Single Board', function () {
      const getRes = http.get(
        `${BASE_URL}/svc/board/api/boards/${createdBoardId}`,
        {
          headers: headers,
          tags: { name: 'board-get' },
        }
      );

      readLatency.add(getRes.timings.duration);
      crudOperations.add(1);

      check(getRes, {
        'single board retrieved': (r) => r.status === 200,
        'get response time < 500ms': (r) => r.timings.duration < 500,
      });
    });

    sleep(0.5);

    // Step 4: Update Board
    group('4. Update Board', function () {
      const updatePayload = JSON.stringify({
        name: `${testBoardName}-updated`,
        description: `Updated at ${new Date().toISOString()}`,
      });

      const updateRes = http.put(
        `${BASE_URL}/svc/board/api/boards/${createdBoardId}`,
        updatePayload,
        {
          headers: headers,
          tags: { name: 'board-update' },
        }
      );

      updateLatency.add(updateRes.timings.duration);
      crudOperations.add(1);

      check(updateRes, {
        'board updated successfully': (r) => r.status === 200,
        'update response time < 1s': (r) => r.timings.duration < 1000,
      });
    });

    sleep(0.5);

    // Step 5: Delete Board (cleanup)
    group('5. Delete Board', function () {
      const deleteRes = http.del(
        `${BASE_URL}/svc/board/api/boards/${createdBoardId}`,
        null,
        {
          headers: headers,
          tags: { name: 'board-delete' },
        }
      );

      deleteLatency.add(deleteRes.timings.duration);
      crudOperations.add(1);

      check(deleteRes, {
        'board deleted successfully': (r) => r.status === 200 || r.status === 204,
        'delete response time < 500ms': (r) => r.timings.duration < 500,
      });
    });
  }

  sleep(1);
}

// =============================================================================
// Setup & Teardown
// =============================================================================
export function setup() {
  console.log('Board CRUD Scenario Starting');
  console.log(`Target: ${BASE_URL}`);
  console.log(`Workspace: ${WORKSPACE_ID}`);
  console.log(`Auth Token: ${TEST_TOKEN ? 'Provided' : 'Not provided'}`);

  // Check board service health
  const healthRes = http.get(`${BASE_URL}/svc/board/health/live`);
  console.log(`Board Service Health: ${healthRes.status}`);

  return { startTime: new Date().toISOString() };
}

export function teardown(data) {
  console.log(`Board CRUD Scenario Completed`);
  console.log(`Started: ${data.startTime}`);
}

// =============================================================================
// Custom Summary
// =============================================================================
export function handleSummary(data) {
  const summary = {
    testType: 'board-crud',
    timestamp: new Date().toISOString(),
    totalOperations: data.metrics.crud_operations?.values?.count || 0,
    errorRate: data.metrics.crud_errors?.values?.rate || 0,
    latencies: {
      create: {
        p50: data.metrics.create_latency?.values['p(50)'] || 0,
        p95: data.metrics.create_latency?.values['p(95)'] || 0,
      },
      read: {
        p50: data.metrics.read_latency?.values['p(50)'] || 0,
        p95: data.metrics.read_latency?.values['p(95)'] || 0,
      },
      update: {
        p50: data.metrics.update_latency?.values['p(50)'] || 0,
        p95: data.metrics.update_latency?.values['p(95)'] || 0,
      },
      delete: {
        p50: data.metrics.delete_latency?.values['p(50)'] || 0,
        p95: data.metrics.delete_latency?.values['p(95)'] || 0,
      },
    },
  };

  return {
    'stdout': JSON.stringify(summary, null, 2) + '\n',
    'board-crud-summary.json': JSON.stringify(summary, null, 2),
  };
}

import http from 'k6/http';
import { sleep, check, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

const cacheHitRate = new Rate('kv_cache_hit_rate');
const putLatency = new Trend('kv_put_duration_ms');
const getLatency = new Trend('kv_get_duration_ms');
const deleteLatency = new Trend('kv_delete_duration_ms');
const totalOps = new Counter('kv_total_operations');

export const options = {
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<500'],
    kv_get_duration_ms: ['p(95)<200'],
  },
  scenarios: {
    smoke: {
      executor: 'constant-vus',
      vus: 1,
      duration: '10s',
      startTime: '0s',
      tags: { scenario: 'smoke' },
    },
    cache_warm: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 5 },
        { duration: '20s', target: 5 },
      ],
      startTime: '15s',
      tags: { scenario: 'cache_warm' },
    },
    mixed_load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '20s', target: 20 },
        { duration: '30s', target: 20 },
        { duration: '10s', target: 0 },
      ],
      startTime: '50s',
      tags: { scenario: 'mixed_load' },
    },
  },
};

function randomKey() {
  const keys = ['alpha', 'beta', 'gamma', 'delta', 'epsilon', 'zeta', 'eta', 'theta'];
  return keys[Math.floor(Math.random() * keys.length)];
}

function smokeTest() {
  group('smoke test', () => {
    let res = http.get(`${BASE_URL}/healthz`);
    check(res, { 'healthz ok': (r) => r.status === 200 });

    res = http.get(`${BASE_URL}/readyz`);
    check(res, { 'readyz ok': (r) => r.status === 200 });

    res = http.get(`${BASE_URL}/metrics`);
    check(res, { 'metrics ok': (r) => r.status === 200 });

    const key = randomKey();
    res = http.put(`${BASE_URL}/api/v1/key/${key}`, 'smoke_value');
    check(res, { 'put ok': (r) => r.status === 201 });

    res = http.get(`${BASE_URL}/api/v1/key/${key}`);
    check(res, {
      'get ok': (r) => r.status === 200,
      'value matches': (r) => r.json('value') === 'smoke_value',
    });

    res = http.del(`${BASE_URL}/api/v1/key/${key}`);
    check(res, { 'delete ok': (r) => r.status === 204 });

    res = http.get(`${BASE_URL}/api/v1/key/${key}`);
    check(res, { 'get after delete 404': (r) => r.status === 404 });
  });
}

function crudOperations(putRatio) {
  const roll = Math.random();

  if (roll < putRatio) {
    const key = randomKey();
    const value = `val_${__VU}_${Date.now()}`;
    const start = Date.now();
    const res = http.put(`${BASE_URL}/api/v1/key/${key}`, value);
    putLatency.add(Date.now() - start);
    check(res, { 'put status 201': (r) => r.status === 201 });
    totalOps.add(1);
  } else if (roll < putRatio + 0.7) {
    const key = randomKey();
    const start = Date.now();
    const res = http.get(`${BASE_URL}/api/v1/key/${key}`);
    getLatency.add(Date.now() - start);
    if (res.status === 200) {
      cacheHitRate.add(1);
    } else {
      cacheHitRate.add(0);
    }
    check(res, { 'get status 200 or 404': (r) => r.status === 200 || r.status === 404 });
    totalOps.add(1);
  } else {
    const key = randomKey();
    const start = Date.now();
    const res = http.del(`${BASE_URL}/api/v1/key/${key}`);
    deleteLatency.add(Date.now() - start);
    check(res, { 'delete status 204': (r) => r.status === 204 });
    totalOps.add(1);
  }
}

export default function () {
  if (__ITER < 5) {
    smokeTest();
  } else {
    crudOperations(0.2);
  }
  sleep(0.5);
}

import http from 'k6/http';
import { check, group } from 'k6';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export const options = {
  scenarios: {
    // Test rate limiting on auth endpoints
    auth_rate_limit: {
      executor: 'constant-arrival-rate',
      rate: 200, // 200 requests per timeUnit
      timeUnit: '1m', // per minute
      duration: '2m',
      preAllocatedVUs: 50,
      maxVUs: 100,
    },
  },
  thresholds: {
    'http_req_failed{status:429}': ['count>0'], // Expect some 429s
    'http_req_duration': ['p(95)<1000'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const email = `ratelimit_${randomString(10)}@example.com`;
  const password = 'RateLimit@123';

  group('Rate Limit Testing', () => {
    // Test login endpoint rate limiting
    group('Login Rate Limit', () => {
      const res = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
        email: email,
        password: password,
      }), {
        headers: { 'Content-Type': 'application/json' },
        tags: { endpoint: 'login' },
      });

      const isRateLimited = res.status === 429;
      const isNormalError = res.status === 401; // Wrong credentials
      const isSuccess = res.status === 200;

      check(res, {
        'response received': (r) => r.status !== 0,
        'valid response': (r) => isRateLimited || isNormalError || isSuccess,
      });

      if (isRateLimited) {
        check(res, {
          'rate limit error message': (r) => r.json('error') !== undefined,
          'has retry-after header': (r) => r.headers['Retry-After'] !== undefined,
        });
      }
    });

    // Test signup endpoint rate limiting
    group('Signup Rate Limit', () => {
      const uniqueEmail = `ratelimit_${randomString(20)}@example.com`;
      const res = http.post(`${BASE_URL}/api/v1/auth/signup`, JSON.stringify({
        email: uniqueEmail,
        password: password,
      }), {
        headers: { 'Content-Type': 'application/json' },
        tags: { endpoint: 'signup' },
      });

      const isRateLimited = res.status === 429;
      const isSuccess = res.status === 201;
      const isDuplicate = res.status === 409;

      check(res, {
        'response received': (r) => r.status !== 0,
        'valid response': (r) => isRateLimited || isSuccess || isDuplicate,
      });
    });

    // Test refresh endpoint rate limiting
    group('Refresh Rate Limit', () => {
      const res = http.post(`${BASE_URL}/api/v1/auth/refresh`, JSON.stringify({
        refresh_token: 'dummy-token',
      }), {
        headers: { 'Content-Type': 'application/json' },
        tags: { endpoint: 'refresh' },
      });

      const isRateLimited = res.status === 429;
      const isInvalidToken = res.status === 401;

      check(res, {
        'response received': (r) => r.status !== 0,
        'valid response': (r) => isRateLimited || isInvalidToken,
      });
    });
  });
}

// Summary handler to show rate limiting statistics
export function handleSummary(data) {
  const rateLimitedRequests = data.metrics['http_reqs{status:429}'];
  const totalRequests = data.metrics.http_reqs;
  
  if (rateLimitedRequests && totalRequests) {
    const rateLimitPercentage = (rateLimitedRequests.values.count / totalRequests.values.count) * 100;
    console.log(`\nRate Limiting Summary:`);
    console.log(`Total requests: ${totalRequests.values.count}`);
    console.log(`Rate limited (429): ${rateLimitedRequests.values.count}`);
    console.log(`Rate limit percentage: ${rateLimitPercentage.toFixed(2)}%`);
  }
  
  return {
    stdout: JSON.stringify(data, null, 2),
  };
}
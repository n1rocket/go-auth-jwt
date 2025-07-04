import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Test configuration
export const options = {
  stages: [
    { duration: '10s', target: 5 },   // Ramp up to 5 users
    { duration: '30s', target: 10 },  // Stay at 10 users
    { duration: '10s', target: 0 },   // Ramp down to 0 users
  ],
  thresholds: {
    http_req_failed: ['rate<0.1'],   // Error rate should be below 10%
    http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
  },
};

// Base URL configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Helper function to generate unique emails
function generateEmail() {
  return `user_${randomString(10)}@test.com`;
}

// Main test scenario
export default function () {
  const email = generateEmail();
  const password = 'Test@Password123';
  let accessToken = '';
  let refreshToken = '';

  // Test 1: Health check endpoint
  let res = http.get(`${BASE_URL}/health`);
  check(res, {
    'health check status is 200': (r) => r.status === 200,
    'health check returns ok': (r) => r.json('status') === 'ok',
  });

  // Test 2: Ready check endpoint
  res = http.get(`${BASE_URL}/ready`);
  check(res, {
    'ready check status is 200': (r) => r.status === 200,
    'ready check returns ready': (r) => r.json('status') === 'ready',
    'ready check has services': (r) => r.json('services') !== null,
  });

  // Test 3: User signup
  const signupPayload = JSON.stringify({
    email: email,
    password: password,
  });

  res = http.post(`${BASE_URL}/api/v1/auth/signup`, signupPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'signup status is 201': (r) => r.status === 201,
    'signup returns access token': (r) => r.json('access_token') !== undefined,
    'signup returns refresh token': (r) => r.json('refresh_token') !== undefined,
    'signup returns user data': (r) => r.json('user') !== undefined,
    'signup returns correct email': (r) => r.json('user.email') === email,
  });

  if (res.status === 201) {
    accessToken = res.json('access_token');
    refreshToken = res.json('refresh_token');
  }

  sleep(1);

  // Test 4: User login
  const loginPayload = JSON.stringify({
    email: email,
    password: password,
  });

  res = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'login status is 200': (r) => r.status === 200,
    'login returns access token': (r) => r.json('access_token') !== undefined,
    'login returns refresh token': (r) => r.json('refresh_token') !== undefined,
    'login returns user data': (r) => r.json('user') !== undefined,
  });

  if (res.status === 200) {
    accessToken = res.json('access_token');
    refreshToken = res.json('refresh_token');
  }

  // Test 5: Get current user (protected endpoint)
  res = http.get(`${BASE_URL}/api/v1/auth/me`, {
    headers: { 
      'Authorization': `Bearer ${accessToken}`,
    },
  });

  check(res, {
    'get user status is 200': (r) => r.status === 200,
    'get user returns correct email': (r) => r.json('email') === email,
    'get user returns id': (r) => r.json('id') !== undefined,
  });

  // Test 6: Refresh token
  const refreshPayload = JSON.stringify({
    refresh_token: refreshToken,
  });

  res = http.post(`${BASE_URL}/api/v1/auth/refresh`, refreshPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'refresh status is 200': (r) => r.status === 200,
    'refresh returns new access token': (r) => r.json('access_token') !== undefined,
    'refresh returns new refresh token': (r) => r.json('refresh_token') !== undefined,
  });

  if (res.status === 200) {
    accessToken = res.json('access_token');
    refreshToken = res.json('refresh_token');
  }

  // Test 7: Email verification (simulate - this would normally come from email)
  const verifyPayload = JSON.stringify({
    token: 'dummy-verification-token',
  });

  res = http.post(`${BASE_URL}/api/v1/auth/verify-email`, verifyPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  // This will fail with dummy token, but we're testing the endpoint exists
  check(res, {
    'verify email endpoint exists': (r) => r.status !== 404,
  });

  // Test 8: Logout
  res = http.post(`${BASE_URL}/api/v1/auth/logout`, JSON.stringify({
    refresh_token: refreshToken,
  }), {
    headers: { 
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${accessToken}`,
    },
  });

  check(res, {
    'logout status is 200': (r) => r.status === 200,
    'logout returns success message': (r) => r.json('message') !== undefined,
  });

  // Test 9: Logout all devices
  // First login again to get new tokens
  res = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (res.status === 200) {
    accessToken = res.json('access_token');
    refreshToken = res.json('refresh_token');

    res = http.post(`${BASE_URL}/api/v1/auth/logout-all`, null, {
      headers: { 
        'Authorization': `Bearer ${accessToken}`,
      },
    });

    check(res, {
      'logout all status is 200': (r) => r.status === 200,
      'logout all returns success message': (r) => r.json('message') !== undefined,
    });
  }

  sleep(1);
}

// Additional test scenarios
export function testErrorScenarios() {
  // Test invalid login
  const invalidLoginPayload = JSON.stringify({
    email: 'nonexistent@test.com',
    password: 'wrongpassword',
  });

  let res = http.post(`${BASE_URL}/api/v1/auth/login`, invalidLoginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'invalid login returns 401': (r) => r.status === 401,
    'invalid login returns error': (r) => r.json('error') !== undefined,
  });

  // Test signup with invalid data
  const invalidSignupPayload = JSON.stringify({
    email: 'invalid-email',
    password: 'short',
  });

  res = http.post(`${BASE_URL}/api/v1/auth/signup`, invalidSignupPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'invalid signup returns 400': (r) => r.status === 400,
    'invalid signup returns validation error': (r) => r.json('error') !== undefined,
  });

  // Test protected endpoint without auth
  res = http.get(`${BASE_URL}/api/v1/auth/me`);

  check(res, {
    'unauthorized request returns 401': (r) => r.status === 401,
  });

  // Test with invalid token
  res = http.get(`${BASE_URL}/api/v1/auth/me`, {
    headers: { 
      'Authorization': 'Bearer invalid-token',
    },
  });

  check(res, {
    'invalid token returns 401': (r) => r.status === 401,
  });
}

// Stress test for login endpoint
export function stressTestLogin() {
  const email = `stress_${randomString(10)}@test.com`;
  const password = 'StressTest@123';

  // First create a user
  const signupPayload = JSON.stringify({
    email: email,
    password: password,
  });

  http.post(`${BASE_URL}/api/v1/auth/signup`, signupPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  // Now stress test login
  const loginPayload = JSON.stringify({
    email: email,
    password: password,
  });

  const res = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'login stress test succeeds': (r) => r.status === 200,
  });
}
import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Single user test to avoid rate limiting
export const options = {
  vus: 1,
  duration: '30s',
  thresholds: {
    http_req_failed: ['rate<0.1'],
    http_req_duration: ['p(95)<500'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const email = `user_${randomString(10)}@test.com`;
  const password = 'Test@Password123';
  let accessToken = '';
  let refreshToken = '';

  console.log(`Testing with user: ${email}`);

  // Test 1: Health check
  let res = http.get(`${BASE_URL}/health`);
  console.log(`Health check: ${res.status}`);
  check(res, {
    'health check status is 200': (r) => r.status === 200,
  });

  // Test 2: Ready check
  res = http.get(`${BASE_URL}/ready`);
  console.log(`Ready check: ${res.status}`);
  check(res, {
    'ready check status is 200': (r) => r.status === 200,
  });

  // Wait to avoid rate limiting
  sleep(2);

  // Test 3: User signup
  const signupPayload = JSON.stringify({
    email: email,
    password: password,
  });

  res = http.post(`${BASE_URL}/api/v1/auth/signup`, signupPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  console.log(`Signup status: ${res.status}`);
  if (res.status !== 201) {
    console.log(`Signup failed: ${res.body}`);
  }

  const signupSuccess = check(res, {
    'signup status is 201': (r) => r.status === 201,
  });

  if (signupSuccess) {
    const body = res.json();
    accessToken = body.access_token;
    refreshToken = body.refresh_token;
    console.log('Signup successful, tokens received');
  }

  // Wait to avoid rate limiting
  sleep(2);

  // Test 4: User login
  const loginPayload = JSON.stringify({
    email: email,
    password: password,
  });

  res = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  console.log(`Login status: ${res.status}`);
  const loginSuccess = check(res, {
    'login status is 200': (r) => r.status === 200,
  });

  if (loginSuccess) {
    const body = res.json();
    accessToken = body.access_token;
    refreshToken = body.refresh_token;
    console.log('Login successful');
  }

  // Wait to avoid rate limiting
  sleep(2);

  // Test 5: Get current user
  res = http.get(`${BASE_URL}/api/v1/auth/me`, {
    headers: { 
      'Authorization': `Bearer ${accessToken}`,
    },
  });

  console.log(`Get user status: ${res.status}`);
  check(res, {
    'get user status is 200': (r) => r.status === 200,
  });

  // Wait to avoid rate limiting
  sleep(2);

  // Test 6: Refresh token
  const refreshPayload = JSON.stringify({
    refresh_token: refreshToken,
  });

  res = http.post(`${BASE_URL}/api/v1/auth/refresh`, refreshPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  console.log(`Refresh status: ${res.status}`);
  check(res, {
    'refresh status is 200': (r) => r.status === 200,
  });

  // Wait to avoid rate limiting
  sleep(2);

  // Test 7: Logout
  res = http.post(`${BASE_URL}/api/v1/auth/logout`, JSON.stringify({
    refresh_token: refreshToken,
  }), {
    headers: { 
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${accessToken}`,
    },
  });

  console.log(`Logout status: ${res.status}`);
  check(res, {
    'logout status is 200': (r) => r.status === 200,
  });

  // Test error scenarios
  sleep(2);

  // Test invalid login
  const invalidLoginPayload = JSON.stringify({
    email: 'nonexistent@test.com',
    password: 'wrongpassword',
  });

  res = http.post(`${BASE_URL}/api/v1/auth/login`, invalidLoginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  console.log(`Invalid login status: ${res.status}`);
  check(res, {
    'invalid login returns 401': (r) => r.status === 401,
  });

  sleep(5); // Longer sleep between iterations
}
import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export const options = {
  scenarios: {
    // Scenario 1: Smoke test - minimal load
    smoke: {
      executor: 'constant-vus',
      vus: 1,
      duration: '1m',
    },
    // Scenario 2: Load test - normal load
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 10 },
        { duration: '1m', target: 20 },
        { duration: '30s', target: 0 },
      ],
      startTime: '1m',
    },
    // Scenario 3: Stress test - beyond normal load
    stress: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 50 },
        { duration: '1m', target: 100 },
        { duration: '30s', target: 0 },
      ],
      startTime: '3m',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.1'],
    http_req_duration: ['p(95)<1000', 'p(99)<2000'],
    'http_req_duration{scenario:smoke}': ['p(95)<500'],
    'http_req_duration{scenario:load}': ['p(95)<800'],
    'http_req_duration{scenario:stress}': ['p(95)<1500'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data generator
class TestUser {
  constructor() {
    this.email = `test_${randomString(10)}@example.com`;
    this.password = `Pass@${randomString(8)}123`;
    this.accessToken = null;
    this.refreshToken = null;
  }
}

// API client helper
class AuthAPI {
  constructor(baseUrl) {
    this.baseUrl = baseUrl;
  }

  signup(email, password) {
    return http.post(`${this.baseUrl}/api/v1/auth/signup`, JSON.stringify({
      email: email,
      password: password,
    }), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'signup' },
    });
  }

  login(email, password) {
    return http.post(`${this.baseUrl}/api/v1/auth/login`, JSON.stringify({
      email: email,
      password: password,
    }), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'login' },
    });
  }

  refresh(refreshToken) {
    return http.post(`${this.baseUrl}/api/v1/auth/refresh`, JSON.stringify({
      refresh_token: refreshToken,
    }), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'refresh' },
    });
  }

  getMe(accessToken) {
    return http.get(`${this.baseUrl}/api/v1/auth/me`, {
      headers: { 'Authorization': `Bearer ${accessToken}` },
      tags: { name: 'getMe' },
    });
  }

  logout(accessToken, refreshToken) {
    return http.post(`${this.baseUrl}/api/v1/auth/logout`, JSON.stringify({
      refresh_token: refreshToken,
    }), {
      headers: { 
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${accessToken}`,
      },
      tags: { name: 'logout' },
    });
  }

  logoutAll(accessToken) {
    return http.post(`${this.baseUrl}/api/v1/auth/logout-all`, null, {
      headers: { 'Authorization': `Bearer ${accessToken}` },
      tags: { name: 'logoutAll' },
    });
  }

  verifyEmail(token) {
    return http.post(`${this.baseUrl}/api/v1/auth/verify-email`, JSON.stringify({
      token: token,
    }), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'verifyEmail' },
    });
  }

  health() {
    return http.get(`${this.baseUrl}/health`, {
      tags: { name: 'health' },
    });
  }

  ready() {
    return http.get(`${this.baseUrl}/ready`, {
      tags: { name: 'ready' },
    });
  }
}

const api = new AuthAPI(BASE_URL);

export default function () {
  // Health checks
  group('Health Checks', () => {
    const healthRes = api.health();
    check(healthRes, {
      'health check is successful': (r) => r.status === 200,
      'health returns ok status': (r) => r.json('status') === 'ok',
    });

    const readyRes = api.ready();
    check(readyRes, {
      'ready check is successful': (r) => r.status === 200,
      'ready returns ready status': (r) => r.json('status') === 'ready',
      'ready includes database service': (r) => r.json('services.database') === 'healthy',
    });
  });

  // Complete user journey
  group('User Journey', () => {
    const user = new TestUser();

    // Signup
    group('Signup', () => {
      const res = api.signup(user.email, user.password);
      const success = check(res, {
        'signup successful': (r) => r.status === 201,
        'returns access token': (r) => !!r.json('access_token'),
        'returns refresh token': (r) => !!r.json('refresh_token'),
        'returns user object': (r) => !!r.json('user'),
        'user email matches': (r) => r.json('user.email') === user.email,
      });

      if (success) {
        user.accessToken = res.json('access_token');
        user.refreshToken = res.json('refresh_token');
      }
    });

    sleep(1);

    // Login
    group('Login', () => {
      const res = api.login(user.email, user.password);
      const success = check(res, {
        'login successful': (r) => r.status === 200,
        'returns tokens': (r) => !!r.json('access_token') && !!r.json('refresh_token'),
      });

      if (success) {
        user.accessToken = res.json('access_token');
        user.refreshToken = res.json('refresh_token');
      }
    });

    // Get current user
    group('Get Current User', () => {
      const res = api.getMe(user.accessToken);
      check(res, {
        'get me successful': (r) => r.status === 200,
        'returns correct email': (r) => r.json('email') === user.email,
        'returns user id': (r) => !!r.json('id'),
      });
    });

    // Refresh token
    group('Refresh Token', () => {
      const res = api.refresh(user.refreshToken);
      const success = check(res, {
        'refresh successful': (r) => r.status === 200,
        'returns new tokens': (r) => !!r.json('access_token') && !!r.json('refresh_token'),
      });

      if (success) {
        user.accessToken = res.json('access_token');
        user.refreshToken = res.json('refresh_token');
      }
    });

    // Logout
    group('Logout', () => {
      const res = api.logout(user.accessToken, user.refreshToken);
      check(res, {
        'logout successful': (r) => r.status === 200,
        'returns success message': (r) => !!r.json('message'),
      });
    });

    sleep(1);
  });

  // Error scenarios
  group('Error Scenarios', () => {
    // Invalid login
    group('Invalid Login', () => {
      const res = api.login('nonexistent@example.com', 'wrongpassword');
      check(res, {
        'returns 401': (r) => r.status === 401,
        'returns error message': (r) => !!r.json('error'),
      });
    });

    // Invalid signup
    group('Invalid Signup', () => {
      const res = api.signup('invalid-email', 'short');
      check(res, {
        'returns 400': (r) => r.status === 400,
        'returns validation error': (r) => !!r.json('error'),
      });
    });

    // Unauthorized access
    group('Unauthorized Access', () => {
      const res = api.getMe('invalid-token');
      check(res, {
        'returns 401': (r) => r.status === 401,
        'returns unauthorized error': (r) => !!r.json('error'),
      });
    });

    // Missing auth header
    group('Missing Auth Header', () => {
      const res = http.get(`${BASE_URL}/api/v1/auth/me`);
      check(res, {
        'returns 401': (r) => r.status === 401,
      });
    });
  });

  // Edge cases
  group('Edge Cases', () => {
    // Duplicate signup
    group('Duplicate Signup', () => {
      const email = `duplicate_${randomString(10)}@example.com`;
      const password = 'DuplicatePass@123';
      
      // First signup
      api.signup(email, password);
      sleep(0.5);
      
      // Try duplicate
      const res = api.signup(email, password);
      check(res, {
        'duplicate signup returns 409': (r) => r.status === 409,
        'returns conflict error': (r) => !!r.json('error'),
      });
    });

    // Empty request bodies
    group('Empty Request Bodies', () => {
      const res = http.post(`${BASE_URL}/api/v1/auth/login`, '{}', {
        headers: { 'Content-Type': 'application/json' },
      });
      check(res, {
        'empty login returns 400': (r) => r.status === 400,
      });
    });

    // Invalid JSON
    group('Invalid JSON', () => {
      const res = http.post(`${BASE_URL}/api/v1/auth/login`, 'invalid json', {
        headers: { 'Content-Type': 'application/json' },
      });
      check(res, {
        'invalid JSON returns 400': (r) => r.status === 400,
      });
    });
  });

  // Multiple device login scenario
  group('Multiple Device Login', () => {
    const user = new TestUser();
    const tokens = [];

    // Create user
    const signupRes = api.signup(user.email, user.password);
    if (signupRes.status === 201) {
      // Login from multiple "devices"
      for (let i = 0; i < 3; i++) {
        const loginRes = api.login(user.email, user.password);
        if (loginRes.status === 200) {
          tokens.push({
            access: loginRes.json('access_token'),
            refresh: loginRes.json('refresh_token'),
          });
        }
        sleep(0.5);
      }

      // Verify all tokens work
      tokens.forEach((token, index) => {
        const res = api.getMe(token.access);
        check(res, {
          [`device ${index + 1} token works`]: (r) => r.status === 200,
        });
      });

      // Logout from all devices
      if (tokens.length > 0) {
        const res = api.logoutAll(tokens[0].access);
        check(res, {
          'logout all successful': (r) => r.status === 200,
        });

        // Verify all tokens are now invalid
        sleep(1);
        tokens.forEach((token, index) => {
          const res = api.getMe(token.access);
          check(res, {
            [`device ${index + 1} token invalidated`]: (r) => r.status === 401,
          });
        });
      }
    }
  });

  sleep(1);
}
/**
 * JWT Auth Service Client for Node.js
 * 
 * Example usage of the JWT authentication service from a Node.js application
 */

const https = require('https');
const EventEmitter = require('events');

class JWTAuthClient extends EventEmitter {
  constructor(options = {}) {
    super();
    
    this.baseURL = options.baseURL || 'http://localhost:8080';
    this.apiPath = options.apiPath || '/api/v1';
    this.accessToken = null;
    this.refreshToken = null;
    this.refreshTimer = null;
    this.autoRefresh = options.autoRefresh !== false;
  }

  /**
   * Make an HTTP request to the auth service
   */
  async request(method, path, data = null, authenticated = false) {
    const url = new URL(`${this.baseURL}${this.apiPath}${path}`);
    
    const options = {
      method,
      headers: {
        'Content-Type': 'application/json',
      },
    };

    if (authenticated && this.accessToken) {
      options.headers['Authorization'] = `Bearer ${this.accessToken}`;
    }

    return new Promise((resolve, reject) => {
      const req = (url.protocol === 'https:' ? https : require('http')).request(url, options, (res) => {
        let body = '';
        
        res.on('data', (chunk) => {
          body += chunk;
        });
        
        res.on('end', () => {
          try {
            const response = {
              status: res.statusCode,
              headers: res.headers,
              data: body ? JSON.parse(body) : null,
            };
            
            if (res.statusCode >= 200 && res.statusCode < 300) {
              resolve(response);
            } else {
              const error = new Error(response.data?.message || `Request failed with status ${res.statusCode}`);
              error.response = response;
              reject(error);
            }
          } catch (err) {
            reject(err);
          }
        });
      });

      req.on('error', reject);

      if (data) {
        req.write(JSON.stringify(data));
      }

      req.end();
    });
  }

  /**
   * Register a new user
   */
  async signup(email, password) {
    try {
      const response = await this.request('POST', '/auth/signup', { email, password });
      this.emit('signup', { email, success: true });
      return response.data;
    } catch (error) {
      this.emit('signup', { email, success: false, error });
      throw error;
    }
  }

  /**
   * Login with email and password
   */
  async login(email, password) {
    try {
      const response = await this.request('POST', '/auth/login', { email, password });
      const { access_token, refresh_token, expires_in } = response.data;
      
      this.accessToken = access_token;
      this.refreshToken = refresh_token;
      
      // Schedule token refresh
      if (this.autoRefresh && expires_in) {
        this.scheduleTokenRefresh(expires_in);
      }
      
      this.emit('login', { email, success: true });
      return response.data;
    } catch (error) {
      this.emit('login', { email, success: false, error });
      throw error;
    }
  }

  /**
   * Refresh the access token
   */
  async refresh() {
    if (!this.refreshToken) {
      throw new Error('No refresh token available');
    }

    try {
      const response = await this.request('POST', '/auth/refresh', { 
        refresh_token: this.refreshToken 
      });
      
      const { access_token, refresh_token, expires_in } = response.data;
      
      this.accessToken = access_token;
      this.refreshToken = refresh_token;
      
      // Reschedule token refresh
      if (this.autoRefresh && expires_in) {
        this.scheduleTokenRefresh(expires_in);
      }
      
      this.emit('tokenRefreshed', { success: true });
      return response.data;
    } catch (error) {
      this.emit('tokenRefreshed', { success: false, error });
      // Clear tokens on refresh failure
      this.accessToken = null;
      this.refreshToken = null;
      this.clearTokenRefresh();
      throw error;
    }
  }

  /**
   * Logout and revoke the refresh token
   */
  async logout() {
    if (!this.refreshToken) {
      return { message: 'Already logged out' };
    }

    try {
      const response = await this.request('POST', '/auth/logout', {
        refresh_token: this.refreshToken
      }, true);
      
      this.emit('logout', { success: true });
      return response.data;
    } catch (error) {
      this.emit('logout', { success: false, error });
      throw error;
    } finally {
      // Clear tokens regardless of API response
      this.accessToken = null;
      this.refreshToken = null;
      this.clearTokenRefresh();
    }
  }

  /**
   * Logout from all devices
   */
  async logoutAll() {
    try {
      const response = await this.request('POST', '/auth/logout-all', null, true);
      this.emit('logoutAll', { success: true });
      return response.data;
    } catch (error) {
      this.emit('logoutAll', { success: false, error });
      throw error;
    } finally {
      // Clear tokens
      this.accessToken = null;
      this.refreshToken = null;
      this.clearTokenRefresh();
    }
  }

  /**
   * Get current user profile
   */
  async getProfile() {
    const response = await this.request('GET', '/auth/me', null, true);
    return response.data;
  }

  /**
   * Verify email with token
   */
  async verifyEmail(email, token) {
    const response = await this.request('POST', '/auth/verify-email', { email, token });
    this.emit('emailVerified', { email });
    return response.data;
  }

  /**
   * Resend verification email
   */
  async resendVerification() {
    const response = await this.request('POST', '/auth/resend-verification', null, true);
    this.emit('verificationResent');
    return response.data;
  }

  /**
   * Make an authenticated API request
   */
  async authenticatedRequest(method, path, data = null) {
    // Retry with token refresh on 401
    try {
      return await this.request(method, path, data, true);
    } catch (error) {
      if (error.response?.status === 401 && this.refreshToken) {
        // Try to refresh token
        await this.refresh();
        // Retry request
        return await this.request(method, path, data, true);
      }
      throw error;
    }
  }

  /**
   * Schedule automatic token refresh
   */
  scheduleTokenRefresh(expiresIn) {
    this.clearTokenRefresh();
    
    // Refresh 30 seconds before expiration
    const refreshTime = (expiresIn - 30) * 1000;
    
    if (refreshTime > 0) {
      this.refreshTimer = setTimeout(async () => {
        try {
          await this.refresh();
        } catch (error) {
          console.error('Auto-refresh failed:', error);
          this.emit('autoRefreshFailed', { error });
        }
      }, refreshTime);
    }
  }

  /**
   * Clear token refresh timer
   */
  clearTokenRefresh() {
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
      this.refreshTimer = null;
    }
  }

  /**
   * Check if user is authenticated
   */
  isAuthenticated() {
    return !!this.accessToken;
  }

  /**
   * Get stored tokens (for persistence)
   */
  getTokens() {
    return {
      accessToken: this.accessToken,
      refreshToken: this.refreshToken,
    };
  }

  /**
   * Set tokens (for restoration)
   */
  setTokens(accessToken, refreshToken, expiresIn = 3600) {
    this.accessToken = accessToken;
    this.refreshToken = refreshToken;
    
    if (this.autoRefresh && expiresIn) {
      this.scheduleTokenRefresh(expiresIn);
    }
  }
}

// Example usage
if (require.main === module) {
  const client = new JWTAuthClient({
    baseURL: 'http://localhost:8080',
    autoRefresh: true,
  });

  // Listen to events
  client.on('login', ({ email, success }) => {
    console.log(`Login ${success ? 'successful' : 'failed'} for ${email}`);
  });

  client.on('tokenRefreshed', ({ success }) => {
    console.log(`Token refresh ${success ? 'successful' : 'failed'}`);
  });

  // Example flow
  async function example() {
    try {
      // Signup
      console.log('1. Signing up new user...');
      await client.signup('test@example.com', 'SecurePassword123!');
      console.log('   ✓ Signup successful');

      // Login
      console.log('2. Logging in...');
      const loginResult = await client.login('test@example.com', 'SecurePassword123!');
      console.log('   ✓ Login successful');
      console.log(`   Access token: ${loginResult.access_token.substring(0, 20)}...`);

      // Get profile
      console.log('3. Getting user profile...');
      const profile = await client.getProfile();
      console.log('   ✓ Profile retrieved:', profile);

      // Wait for auto-refresh (if needed)
      console.log('4. Waiting for token refresh...');
      await new Promise(resolve => setTimeout(resolve, 5000));

      // Logout
      console.log('5. Logging out...');
      await client.logout();
      console.log('   ✓ Logout successful');

    } catch (error) {
      console.error('Error:', error.message);
      if (error.response) {
        console.error('Response:', error.response.data);
      }
    }
  }

  // Run example
  example();
}

module.exports = JWTAuthClient;
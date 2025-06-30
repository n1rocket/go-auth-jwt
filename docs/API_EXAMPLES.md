# API Examples and Integration Guide

## Table of Contents

1. [Quick Start](#quick-start)
2. [Authentication Flow](#authentication-flow)
3. [Client Examples](#client-examples)
4. [Error Handling](#error-handling)
5. [Security Best Practices](#security-best-practices)

## Quick Start

### 1. Start the Server

```bash
# Set up environment variables
export JWT_SECRET="your-secret-key-min-32-chars-long"
export DATABASE_DSN="postgres://user:password@localhost/authdb?sslmode=disable"
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT="587"
export SMTP_USERNAME="your-email@gmail.com"
export SMTP_PASSWORD="your-app-password"

# Run the server
go run cmd/api/main.go
```

### 2. Register a New User

```bash
curl -X POST http://localhost:8080/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "SecurePass123"
  }'
```

### 3. Verify Email

Check your email for the verification link, or use the token directly:

```bash
curl -X POST http://localhost:8080/api/v1/auth/verify-email \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "token": "verification-token-from-email"
  }'
```

### 4. Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "SecurePass123"
  }' | jq '.'
```

Save the `access_token` and `refresh_token` from the response.

## Authentication Flow

### Complete Authentication Flow Diagram

```
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
│ Client  │     │   API   │     │Database │     │  Email  │
└────┬────┘     └────┬────┘     └────┬────┘     └────┬────┘
     │               │               │               │
     │  1. Signup    │               │               │
     │──────────────>│               │               │
     │               │  2. Create    │               │
     │               │──────────────>│               │
     │               │               │               │
     │               │  3. Send Email│               │
     │               │──────────────────────────────>│
     │               │               │               │
     │  4. Response  │               │               │
     │<──────────────│               │               │
     │               │               │               │
     │  5. Verify    │               │               │
     │──────────────>│               │               │
     │               │  6. Update    │               │
     │               │──────────────>│               │
     │               │               │               │
     │  7. Login     │               │               │
     │──────────────>│               │               │
     │               │  8. Validate  │               │
     │               │──────────────>│               │
     │               │               │               │
     │  9. Tokens    │               │               │
     │<──────────────│               │               │
```

## Client Examples

### JavaScript/TypeScript Client

```typescript
class AuthClient {
  private baseURL = 'http://localhost:8080/api/v1';
  private accessToken: string | null = null;
  private refreshToken: string | null = null;

  async signup(email: string, password: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/auth/signup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message);
    }

    const data = await response.json();
    console.log('Signup successful:', data.message);
  }

  async login(email: string, password: string): Promise<void> {
    const response = await fetch(`${this.baseURL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password })
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.message);
    }

    const data = await response.json();
    this.accessToken = data.access_token;
    this.refreshToken = data.refresh_token;
    
    // Set up automatic token refresh
    this.scheduleTokenRefresh(data.expires_in);
  }

  async refreshAccessToken(): Promise<void> {
    if (!this.refreshToken) {
      throw new Error('No refresh token available');
    }

    const response = await fetch(`${this.baseURL}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: this.refreshToken })
    });

    if (!response.ok) {
      // Refresh token expired, need to login again
      this.accessToken = null;
      this.refreshToken = null;
      throw new Error('Session expired, please login again');
    }

    const data = await response.json();
    this.accessToken = data.access_token;
    this.refreshToken = data.refresh_token;
    
    this.scheduleTokenRefresh(data.expires_in);
  }

  private scheduleTokenRefresh(expiresIn: number): void {
    // Refresh token 30 seconds before expiration
    const refreshTime = (expiresIn - 30) * 1000;
    setTimeout(() => {
      this.refreshAccessToken().catch(console.error);
    }, refreshTime);
  }

  async makeAuthenticatedRequest(endpoint: string, options: RequestInit = {}): Promise<Response> {
    if (!this.accessToken) {
      throw new Error('Not authenticated');
    }

    const response = await fetch(`${this.baseURL}${endpoint}`, {
      ...options,
      headers: {
        ...options.headers,
        'Authorization': `Bearer ${this.accessToken}`
      }
    });

    if (response.status === 401) {
      // Token expired, try to refresh
      await this.refreshAccessToken();
      
      // Retry request with new token
      return fetch(`${this.baseURL}${endpoint}`, {
        ...options,
        headers: {
          ...options.headers,
          'Authorization': `Bearer ${this.accessToken}`
        }
      });
    }

    return response;
  }

  async getCurrentUser(): Promise<any> {
    const response = await this.makeAuthenticatedRequest('/auth/me');
    return response.json();
  }

  async logout(): Promise<void> {
    if (!this.refreshToken) return;

    await this.makeAuthenticatedRequest('/auth/logout', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: this.refreshToken })
    });

    this.accessToken = null;
    this.refreshToken = null;
  }
}

// Usage example
const client = new AuthClient();

async function example() {
  try {
    // Signup
    await client.signup('user@example.com', 'SecurePass123');
    
    // After email verification, login
    await client.login('user@example.com', 'SecurePass123');
    
    // Get user info
    const user = await client.getCurrentUser();
    console.log('Current user:', user);
    
    // Logout
    await client.logout();
  } catch (error) {
    console.error('Error:', error);
  }
}
```

### Python Client

```python
import requests
import time
from typing import Optional, Dict, Any
from threading import Timer

class AuthClient:
    def __init__(self, base_url: str = "http://localhost:8080/api/v1"):
        self.base_url = base_url
        self.access_token: Optional[str] = None
        self.refresh_token: Optional[str] = None
        self.refresh_timer: Optional[Timer] = None

    def signup(self, email: str, password: str) -> Dict[str, Any]:
        response = requests.post(
            f"{self.base_url}/auth/signup",
            json={"email": email, "password": password}
        )
        response.raise_for_status()
        return response.json()

    def login(self, email: str, password: str) -> None:
        response = requests.post(
            f"{self.base_url}/auth/login",
            json={"email": email, "password": password}
        )
        response.raise_for_status()
        
        data = response.json()
        self.access_token = data["access_token"]
        self.refresh_token = data["refresh_token"]
        
        # Schedule token refresh
        self._schedule_refresh(data["expires_in"])

    def refresh_access_token(self) -> None:
        if not self.refresh_token:
            raise Exception("No refresh token available")

        response = requests.post(
            f"{self.base_url}/auth/refresh",
            json={"refresh_token": self.refresh_token}
        )
        
        if response.status_code != 200:
            self.access_token = None
            self.refresh_token = None
            raise Exception("Session expired, please login again")
        
        data = response.json()
        self.access_token = data["access_token"]
        self.refresh_token = data["refresh_token"]
        
        self._schedule_refresh(data["expires_in"])

    def _schedule_refresh(self, expires_in: int) -> None:
        if self.refresh_timer:
            self.refresh_timer.cancel()
        
        # Refresh 30 seconds before expiration
        refresh_time = expires_in - 30
        self.refresh_timer = Timer(refresh_time, self.refresh_access_token)
        self.refresh_timer.start()

    def _make_authenticated_request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        if not self.access_token:
            raise Exception("Not authenticated")

        headers = kwargs.get("headers", {})
        headers["Authorization"] = f"Bearer {self.access_token}"
        kwargs["headers"] = headers

        response = requests.request(method, f"{self.base_url}{endpoint}", **kwargs)
        
        if response.status_code == 401:
            # Token expired, try to refresh
            self.refresh_access_token()
            
            # Retry with new token
            headers["Authorization"] = f"Bearer {self.access_token}"
            response = requests.request(method, f"{self.base_url}{endpoint}", **kwargs)
        
        return response

    def get_current_user(self) -> Dict[str, Any]:
        response = self._make_authenticated_request("GET", "/auth/me")
        response.raise_for_status()
        return response.json()

    def logout(self) -> None:
        if not self.refresh_token:
            return

        self._make_authenticated_request(
            "POST", 
            "/auth/logout",
            json={"refresh_token": self.refresh_token}
        )
        
        self.access_token = None
        self.refresh_token = None
        
        if self.refresh_timer:
            self.refresh_timer.cancel()

# Usage example
if __name__ == "__main__":
    client = AuthClient()
    
    try:
        # Signup
        result = client.signup("user@example.com", "SecurePass123")
        print(f"Signup successful: {result['message']}")
        
        # After email verification, login
        client.login("user@example.com", "SecurePass123")
        
        # Get user info
        user = client.get_current_user()
        print(f"Current user: {user}")
        
        # Logout
        client.logout()
        print("Logged out successfully")
        
    except Exception as e:
        print(f"Error: {e}")
```

### Go Client

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type AuthClient struct {
    BaseURL      string
    AccessToken  string
    RefreshToken string
    client       *http.Client
}

func NewAuthClient(baseURL string) *AuthClient {
    return &AuthClient{
        BaseURL: baseURL,
        client:  &http.Client{Timeout: 10 * time.Second},
    }
}

func (c *AuthClient) Signup(email, password string) error {
    payload := map[string]string{
        "email":    email,
        "password": password,
    }
    
    _, err := c.makeRequest("POST", "/auth/signup", payload, false)
    return err
}

func (c *AuthClient) Login(email, password string) error {
    payload := map[string]string{
        "email":    email,
        "password": password,
    }
    
    resp, err := c.makeRequest("POST", "/auth/login", payload, false)
    if err != nil {
        return err
    }
    
    var result struct {
        AccessToken  string `json:"access_token"`
        RefreshToken string `json:"refresh_token"`
        ExpiresIn    int    `json:"expires_in"`
    }
    
    if err := json.Unmarshal(resp, &result); err != nil {
        return err
    }
    
    c.AccessToken = result.AccessToken
    c.RefreshToken = result.RefreshToken
    
    // Schedule token refresh
    go c.scheduleRefresh(result.ExpiresIn)
    
    return nil
}

func (c *AuthClient) scheduleRefresh(expiresIn int) {
    refreshTime := time.Duration(expiresIn-30) * time.Second
    time.Sleep(refreshTime)
    c.RefreshAccessToken()
}

func (c *AuthClient) RefreshAccessToken() error {
    payload := map[string]string{
        "refresh_token": c.RefreshToken,
    }
    
    resp, err := c.makeRequest("POST", "/auth/refresh", payload, false)
    if err != nil {
        c.AccessToken = ""
        c.RefreshToken = ""
        return err
    }
    
    var result struct {
        AccessToken  string `json:"access_token"`
        RefreshToken string `json:"refresh_token"`
        ExpiresIn    int    `json:"expires_in"`
    }
    
    if err := json.Unmarshal(resp, &result); err != nil {
        return err
    }
    
    c.AccessToken = result.AccessToken
    c.RefreshToken = result.RefreshToken
    
    go c.scheduleRefresh(result.ExpiresIn)
    
    return nil
}

func (c *AuthClient) makeRequest(method, endpoint string, payload interface{}, authenticated bool) ([]byte, error) {
    var body io.Reader
    if payload != nil {
        jsonData, err := json.Marshal(payload)
        if err != nil {
            return nil, err
        }
        body = bytes.NewBuffer(jsonData)
    }
    
    req, err := http.NewRequest(method, c.BaseURL+endpoint, body)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    if authenticated && c.AccessToken != "" {
        req.Header.Set("Authorization", "Bearer "+c.AccessToken)
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    
    if resp.StatusCode >= 400 {
        if resp.StatusCode == 401 && authenticated {
            // Try to refresh token
            if err := c.RefreshAccessToken(); err != nil {
                return nil, err
            }
            // Retry request
            return c.makeRequest(method, endpoint, payload, authenticated)
        }
        
        var errorResp struct {
            Message string `json:"message"`
        }
        json.Unmarshal(respBody, &errorResp)
        return nil, fmt.Errorf("API error: %s", errorResp.Message)
    }
    
    return respBody, nil
}

func (c *AuthClient) GetCurrentUser() (map[string]interface{}, error) {
    resp, err := c.makeRequest("GET", "/auth/me", nil, true)
    if err != nil {
        return nil, err
    }
    
    var user map[string]interface{}
    err = json.Unmarshal(resp, &user)
    return user, err
}

func (c *AuthClient) Logout() error {
    payload := map[string]string{
        "refresh_token": c.RefreshToken,
    }
    
    _, err := c.makeRequest("POST", "/auth/logout", payload, true)
    c.AccessToken = ""
    c.RefreshToken = ""
    return err
}

func main() {
    client := NewAuthClient("http://localhost:8080/api/v1")
    
    // Example usage
    if err := client.Signup("user@example.com", "SecurePass123"); err != nil {
        fmt.Printf("Signup error: %v\n", err)
    }
    
    // After email verification
    if err := client.Login("user@example.com", "SecurePass123"); err != nil {
        fmt.Printf("Login error: %v\n", err)
        return
    }
    
    user, err := client.GetCurrentUser()
    if err != nil {
        fmt.Printf("Get user error: %v\n", err)
        return
    }
    
    fmt.Printf("Current user: %v\n", user)
    
    if err := client.Logout(); err != nil {
        fmt.Printf("Logout error: %v\n", err)
    }
}
```

## Error Handling

### Common Error Scenarios

1. **Invalid Credentials**
```json
{
  "error": "unauthorized",
  "message": "Invalid email or password",
  "code": "INVALID_CREDENTIALS"
}
```

2. **Validation Errors**
```json
{
  "error": "validation_error",
  "message": "Request validation failed",
  "code": "VALIDATION_FAILED",
  "details": {
    "email": "Invalid email format",
    "password": "Password must be at least 8 characters long"
  }
}
```

3. **Rate Limiting**
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests",
  "code": "RATE_LIMIT_EXCEEDED",
  "details": {
    "retry_after": 60
  }
}
```

### Error Handling Best Practices

```javascript
async function handleApiError(response: Response) {
  if (!response.ok) {
    const error = await response.json();
    
    switch (error.code) {
      case 'INVALID_CREDENTIALS':
        // Show login error
        break;
      case 'DUPLICATE_EMAIL':
        // Show email already exists error
        break;
      case 'VALIDATION_FAILED':
        // Show field-specific errors
        for (const [field, message] of Object.entries(error.details || {})) {
          console.error(`${field}: ${message}`);
        }
        break;
      case 'RATE_LIMIT_EXCEEDED':
        // Wait and retry
        const retryAfter = error.details?.retry_after || 60;
        setTimeout(() => {
          // Retry request
        }, retryAfter * 1000);
        break;
      default:
        // Generic error handling
        console.error('API Error:', error.message);
    }
  }
}
```

## Security Best Practices

### 1. Token Storage

**DO:**
- Store tokens in memory for web applications
- Use secure storage (Keychain/Keystore) for mobile apps
- Implement token rotation

**DON'T:**
- Store tokens in localStorage (XSS vulnerable)
- Store tokens in cookies without httpOnly flag
- Log or display tokens

### 2. HTTPS Usage

Always use HTTPS in production:

```javascript
// Development
const API_URL = 'http://localhost:8080/api/v1';

// Production
const API_URL = 'https://api.yourdomain.com/v1';
```

### 3. Token Refresh Strategy

```javascript
class TokenManager {
  private refreshTimer?: NodeJS.Timeout;
  
  setTokens(accessToken: string, refreshToken: string, expiresIn: number) {
    this.accessToken = accessToken;
    this.refreshToken = refreshToken;
    
    // Clear existing timer
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
    }
    
    // Refresh 1 minute before expiration
    const refreshTime = (expiresIn - 60) * 1000;
    this.refreshTimer = setTimeout(() => {
      this.refreshTokens();
    }, refreshTime);
  }
  
  async refreshTokens() {
    try {
      const response = await fetch('/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: this.refreshToken })
      });
      
      if (response.ok) {
        const data = await response.json();
        this.setTokens(data.access_token, data.refresh_token, data.expires_in);
      } else {
        // Handle refresh failure
        this.handleLogout();
      }
    } catch (error) {
      console.error('Token refresh failed:', error);
      this.handleLogout();
    }
  }
  
  handleLogout() {
    this.accessToken = null;
    this.refreshToken = null;
    if (this.refreshTimer) {
      clearTimeout(this.refreshTimer);
    }
    // Redirect to login
  }
}
```

### 4. Request Interceptors

```javascript
// Axios example
axios.interceptors.request.use(
  (config) => {
    const token = tokenManager.getAccessToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

axios.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;
      
      try {
        await tokenManager.refreshTokens();
        originalRequest.headers.Authorization = `Bearer ${tokenManager.getAccessToken()}`;
        return axios(originalRequest);
      } catch (refreshError) {
        tokenManager.handleLogout();
        return Promise.reject(refreshError);
      }
    }
    
    return Promise.reject(error);
  }
);
```

### 5. CORS Configuration

For production, configure specific allowed origins:

```go
// In your server configuration
corsConfig := middleware.CORSConfig{
    AllowedOrigins: []string{
        "https://app.yourdomain.com",
        "https://mobile.yourdomain.com",
    },
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowedHeaders: []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
}
```

## Testing

### Unit Testing Authentication

```javascript
describe('AuthClient', () => {
  let client: AuthClient;
  
  beforeEach(() => {
    client = new AuthClient();
    // Mock fetch
    global.fetch = jest.fn();
  });
  
  test('login stores tokens', async () => {
    const mockResponse = {
      access_token: 'test-access-token',
      refresh_token: 'test-refresh-token',
      expires_in: 900
    };
    
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockResponse
    });
    
    await client.login('test@example.com', 'password');
    
    expect(client.accessToken).toBe('test-access-token');
    expect(client.refreshToken).toBe('test-refresh-token');
  });
  
  test('handles expired token', async () => {
    // First request fails with 401
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: false,
      status: 401
    });
    
    // Refresh succeeds
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        access_token: 'new-access-token',
        refresh_token: 'new-refresh-token',
        expires_in: 900
      })
    });
    
    // Retry succeeds
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ id: '123', email: 'test@example.com' })
    });
    
    const user = await client.getCurrentUser();
    expect(user.id).toBe('123');
  });
});
```

### Integration Testing

```bash
#!/bin/bash
# integration-test.sh

# Start test database
docker run -d --name test-postgres \
  -e POSTGRES_PASSWORD=testpass \
  -e POSTGRES_DB=authdb_test \
  -p 5433:5432 \
  postgres:15

# Wait for database
sleep 5

# Run migrations
migrate -path ./migrations -database "postgres://postgres:testpass@localhost:5433/authdb_test?sslmode=disable" up

# Start test server
DATABASE_DSN="postgres://postgres:testpass@localhost:5433/authdb_test?sslmode=disable" \
JWT_SECRET="test-secret-key-for-testing-only" \
go run cmd/api/main.go &
SERVER_PID=$!

# Wait for server
sleep 2

# Run integration tests
npm test -- --testPathPattern=integration

# Cleanup
kill $SERVER_PID
docker stop test-postgres
docker rm test-postgres
```
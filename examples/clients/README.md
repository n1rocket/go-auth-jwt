# JWT Auth Service Client Examples

This directory contains example client implementations for various programming languages and frameworks, demonstrating how to integrate with the JWT Authentication Service.

## Available Clients

### 1. JavaScript/Node.js Client

A full-featured JavaScript client with automatic token refresh and event handling.

**Features:**
- Promise-based API
- Automatic token refresh
- Event emitter for auth events
- Retry logic for failed requests
- Session persistence

**Usage:**
```javascript
const JWTAuthClient = require('./javascript/auth-client');

const client = new JWTAuthClient({
  baseURL: 'http://localhost:8080',
  autoRefresh: true
});

// Login
await client.login('user@example.com', 'password');

// Get profile
const profile = await client.getProfile();
```

[View Example](javascript/)

### 2. Python Client

A Python client with support for both synchronous usage and Flask integration.

**Features:**
- Sync/async support
- Automatic token refresh
- Session management
- Flask integration example
- Type hints

**Usage:**
```python
from jwt_auth_client import JWTAuthClient

with JWTAuthClient() as client:
    # Login
    client.login("user@example.com", "password")
    
    # Get profile
    profile = client.get_profile()
```

[View Example](python/)

### 3. Go Client

A native Go client with context support and retry capabilities.

**Features:**
- Context-aware API
- Automatic token refresh
- Retry with backoff
- Type-safe responses
- Concurrent-safe

**Usage:**
```go
client := jwtauthclient.NewClient(jwtauthclient.Config{
    BaseURL: "http://localhost:8080",
    AutoRefresh: true,
})

// Login
auth, err := client.Login(ctx, "user@example.com", "password")

// Get profile
profile, err := client.GetProfile(ctx)
```

[View Example](go/)

### 4. React Client

A React context-based authentication system with hooks and components.

**Features:**
- React Context API
- Custom hooks (useAuth)
- Protected route components
- Automatic token management
- TypeScript support

**Usage:**
```tsx
import { useAuth } from './auth-context';

function LoginComponent() {
  const { login, user, error } = useAuth();
  
  const handleLogin = async () => {
    await login(email, password);
  };
}
```

[View Example](react/)

### 5. CLI Client

A command-line interface for interacting with the auth service.

**Features:**
- Interactive prompts
- Secure password input
- Token persistence
- Configuration management
- Multiple commands

**Usage:**
```bash
# Signup
jwt-auth-cli signup

# Login
jwt-auth-cli login

# View profile
jwt-auth-cli profile

# Logout
jwt-auth-cli logout
```

[View Example](cli/)

## Common Integration Patterns

### 1. Token Storage

**Browser (React/JS):**
- Store in memory for security
- Optional localStorage for persistence
- Consider httpOnly cookies

**Server-side (Node.js/Python):**
- Session storage
- Redis for distributed systems
- Encrypted cookies

**Mobile/Desktop (Go/CLI):**
- Secure keychain/keystore
- Encrypted file storage

### 2. Token Refresh Strategy

All clients implement automatic token refresh:

1. Schedule refresh 30 seconds before expiry
2. Retry failed requests after refresh
3. Clear tokens on refresh failure
4. Emit events for monitoring

### 3. Error Handling

```javascript
// JavaScript
try {
  await client.login(email, password);
} catch (error) {
  if (error.response?.status === 401) {
    // Invalid credentials
  } else if (error.response?.status === 429) {
    // Rate limited
  }
}
```

```python
# Python
try:
    client.login(email, password)
except JWTAuthError as e:
    if e.status_code == 401:
        # Invalid credentials
    elif e.status_code == 429:
        # Rate limited
```

### 4. Interceptors/Middleware

**Adding auth headers:**
```javascript
// Axios interceptor
axios.interceptors.request.use(config => {
  config.headers.Authorization = `Bearer ${client.accessToken}`;
  return config;
});
```

**Handling 401 responses:**
```python
# Flask middleware
@app.before_request
def inject_auth():
    if 'tokens' in session:
        g.auth_client.set_tokens(
            session['tokens']['access_token'],
            session['tokens']['refresh_token']
        )
```

## Security Best Practices

1. **Never store tokens in plain text**
   - Use secure storage mechanisms
   - Encrypt sensitive data at rest

2. **Implement token rotation**
   - Refresh tokens before expiry
   - Revoke old tokens after refresh

3. **Use HTTPS in production**
   - All clients default to HTTP for development
   - Always use HTTPS in production

4. **Validate SSL certificates**
   - Don't disable certificate validation
   - Pin certificates for mobile apps

5. **Handle token expiration gracefully**
   - Implement retry logic
   - Clear expired tokens
   - Redirect to login when needed

## Testing

Each client includes examples of:
- Unit tests
- Integration tests
- Mock implementations
- Error scenarios

## Contributing

To add a new client example:

1. Create a new directory with the language/framework name
2. Implement the core authentication methods
3. Add examples and documentation
4. Include error handling and security considerations
5. Submit a pull request

## License

All examples are provided under the MIT License.
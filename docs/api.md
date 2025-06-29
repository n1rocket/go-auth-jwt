# API Documentation

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Most endpoints require authentication via JWT tokens. Include the access token in the Authorization header:

```
Authorization: Bearer <access_token>
```

## Endpoints

### Auth Endpoints

#### POST /auth/signup
Create a new user account.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response (201 Created):**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "User created successfully. Please check your email to verify your account."
}
```

**Error Responses:**
- 400 Bad Request: Invalid email format or weak password
- 409 Conflict: Email already exists

---

#### POST /auth/login
Authenticate and receive tokens.

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "550e8400-e29b-41d4-a716-446655440000",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**Error Responses:**
- 401 Unauthorized: Invalid credentials

---

#### POST /auth/refresh
Get new tokens using refresh token.

**Request Body:**
```json
{
  "refresh_token": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "new-refresh-token",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**Error Responses:**
- 401 Unauthorized: Invalid or expired refresh token

---

#### POST /auth/logout
Logout and revoke refresh token. **Requires authentication.**

**Request Body:**
```json
{
  "refresh_token": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response (200 OK):**
```json
{
  "message": "Logged out successfully"
}
```

---

#### POST /auth/logout-all
Logout from all devices. **Requires authentication.**

**Response (200 OK):**
```json
{
  "message": "Logged out from all devices successfully"
}
```

---

#### POST /auth/verify-email
Verify email address with token.

**Request Body:**
```json
{
  "email": "user@example.com",
  "token": "verification-token-from-email"
}
```

**Response (200 OK):**
```json
{
  "message": "Email verified successfully"
}
```

**Error Responses:**
- 401 Unauthorized: Invalid token

---

#### GET /auth/me
Get current user information. **Requires authentication.**

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "email_verified": true,
  "created_at": "2024-01-01T00:00:00Z"
}
```

---

### Health Check Endpoints

#### GET /health
Basic health check.

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

---

#### GET /ready
Readiness check with service statuses.

**Response (200 OK):**
```json
{
  "status": "ready",
  "services": {
    "database": "ok",
    "auth": "ok"
  }
}
```

## Error Response Format

All errors follow this format:

```json
{
  "error": "error_type",
  "message": "Human readable error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "Additional information"
  }
}
```

## Common Error Codes

- `INVALID_EMAIL`: Email format is invalid
- `WEAK_PASSWORD`: Password doesn't meet requirements
- `DUPLICATE_EMAIL`: Email already exists
- `INVALID_CREDENTIALS`: Email or password is incorrect
- `INVALID_TOKEN`: Token is invalid or expired
- `UNAUTHORIZED`: Authentication required
- `VALIDATION_FAILED`: Request validation failed
- `INTERNAL_ERROR`: Server error

## Rate Limiting

API endpoints are rate limited to prevent abuse:
- Authentication endpoints: 5 requests per minute per IP
- Protected endpoints: 100 requests per minute per user

## Password Requirements

- Minimum 8 characters
- Maximum 72 characters (bcrypt limitation)

## Token Expiration

- Access tokens: 15 minutes
- Refresh tokens: 7 days
- Email verification tokens: 24 hours
# Postman Collection for Go Auth JWT API

This directory contains Postman collections and environments for testing the Go Auth JWT API.

## Files

- `go-auth-jwt.postman_collection.json` - Main collection with all API endpoints
- `go-auth-jwt.postman_environment.json` - Environment variables for local development

## How to Import

1. Open Postman
2. Click on "Import" button in the top left
3. Select both JSON files
4. Click "Import"

## Features

### Automatic Token Management

The collection includes test scripts that automatically:
- Save `accessToken` and `refreshToken` after successful login
- Save `userId` after successful signup
- Use saved tokens for authenticated requests

### Organized Endpoints

The collection is organized into folders:
- **System** - Health checks and system endpoints
- **Authentication** - Signup, login, refresh token, email verification
- **Protected Endpoints** - User profile, logout endpoints
- **Test Scenarios** - Common error scenarios for testing

### Environment Variables

The environment includes:
- `baseUrl` - API base URL (default: http://localhost:8080)
- `userEmail` - Test user email
- `userPassword` - Test user password
- `accessToken` - Automatically populated after login
- `refreshToken` - Automatically populated after login
- `userId` - Automatically populated after signup

## Usage Flow

1. **Sign Up** - Create a new user account
2. **Login** - Authenticate and receive tokens
3. **Get Current User** - Test authenticated endpoint
4. **Refresh Token** - Get new tokens
5. **Logout** - Invalidate refresh token

## Testing Different Scenarios

### Success Flow
1. Run "Sign Up" to create a new user
2. Check MailHog (http://localhost:8025) for verification email
3. Run "Login" with the same credentials
4. Run "Get Current User" to verify authentication works

### Error Testing
Use the "Test Scenarios" folder to test:
- Invalid credentials
- Expired tokens
- Missing authentication
- Invalid token format

## Tips

- After login, the access token is automatically saved and used for subsequent requests
- You can manually update environment variables for different test users
- Check the "Tests" tab in each request to see automatic variable extraction
- Use "Runner" feature to execute the entire collection in sequence

## Rate Limiting

Note that authentication endpoints have rate limiting:
- Signup: 10 requests/hour
- Login: 20 requests/hour
- Refresh: 30 requests/hour

Protected endpoints: 100 requests/minute
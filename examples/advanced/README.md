# Advanced Features Example

This example demonstrates advanced features and patterns for building production-ready applications with the JWT auth service.

## Features Demonstrated

- Email service integration with SMTP
- Worker pool for asynchronous tasks
- Rate limiting middleware
- CORS configuration
- Security headers middleware

## Running the Example

```bash
# From the examples/advanced directory
go run main.go

# Make sure you have MailHog running for email testing:
docker run -d -p 1025:1025 -p 8025:8025 mailhog/mailhog
```

## Components

### 1. Email Service

Shows how to:
- Configure SMTP settings
- Send verification emails
- Send welcome emails
- Use email templates
- Handle email errors

### 2. Worker Pool

Demonstrates:
- Creating a worker pool
- Submitting async tasks
- Handling task errors
- Graceful shutdown
- Queue management

### 3. Rate Limiting

Examples of:
- IP-based rate limiting
- User-based rate limiting
- Path-specific limits
- Custom rate limit keys

### 4. CORS Configuration

Shows how to:
- Allow specific origins
- Configure allowed methods
- Set allowed headers
- Handle preflight requests

### 5. Security Headers

Implements:
- Content Security Policy (CSP)
- X-Frame-Options
- X-Content-Type-Options
- Strict-Transport-Security
- X-XSS-Protection

## Code Structure

The example is organized into separate functions for each feature:
- `emailExample()` - Email service usage
- `workerPoolExample()` - Async task processing
- `rateLimitingExample()` - Rate limit configurations
- `corsExample()` - CORS setup
- `securityHeadersExample()` - Security headers

## Best Practices

1. Always use context for cancellation
2. Implement graceful shutdown
3. Use structured logging
4. Handle errors appropriately
5. Configure timeouts
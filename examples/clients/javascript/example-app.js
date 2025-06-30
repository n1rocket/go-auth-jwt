/**
 * Example Express.js application using JWT Auth Client
 */

const express = require('express');
const session = require('express-session');
const JWTAuthClient = require('./auth-client');

const app = express();
const authClient = new JWTAuthClient({
  baseURL: process.env.AUTH_SERVICE_URL || 'http://localhost:8080',
  autoRefresh: true,
});

// Middleware
app.use(express.json());
app.use(session({
  secret: 'your-session-secret',
  resave: false,
  saveUninitialized: false,
}));

// Store auth client in app locals
app.locals.authClient = authClient;

// Authentication middleware
function requireAuth(req, res, next) {
  if (!req.session.tokens) {
    return res.status(401).json({ error: 'Authentication required' });
  }

  // Restore tokens to client
  authClient.setTokens(
    req.session.tokens.accessToken,
    req.session.tokens.refreshToken
  );

  next();
}

// Routes

/**
 * POST /api/signup
 * Register a new user
 */
app.post('/api/signup', async (req, res) => {
  try {
    const { email, password } = req.body;
    const result = await authClient.signup(email, password);
    res.json(result);
  } catch (error) {
    res.status(error.response?.status || 500).json({
      error: error.message,
      details: error.response?.data,
    });
  }
});

/**
 * POST /api/login
 * Login user
 */
app.post('/api/login', async (req, res) => {
  try {
    const { email, password } = req.body;
    const result = await authClient.login(email, password);
    
    // Store tokens in session
    req.session.tokens = {
      accessToken: result.access_token,
      refreshToken: result.refresh_token,
    };
    
    res.json({
      message: 'Login successful',
      expiresIn: result.expires_in,
    });
  } catch (error) {
    res.status(error.response?.status || 500).json({
      error: error.message,
      details: error.response?.data,
    });
  }
});

/**
 * POST /api/logout
 * Logout user
 */
app.post('/api/logout', requireAuth, async (req, res) => {
  try {
    await authClient.logout();
    req.session.destroy();
    res.json({ message: 'Logout successful' });
  } catch (error) {
    // Still destroy session even if API call fails
    req.session.destroy();
    res.json({ message: 'Logout successful (local)' });
  }
});

/**
 * GET /api/profile
 * Get user profile
 */
app.get('/api/profile', requireAuth, async (req, res) => {
  try {
    const profile = await authClient.getProfile();
    res.json(profile);
  } catch (error) {
    if (error.response?.status === 401) {
      // Token might be expired, try to refresh
      try {
        await authClient.refresh();
        // Update session with new tokens
        req.session.tokens = authClient.getTokens();
        // Retry request
        const profile = await authClient.getProfile();
        res.json(profile);
      } catch (refreshError) {
        req.session.destroy();
        res.status(401).json({ error: 'Session expired' });
      }
    } else {
      res.status(error.response?.status || 500).json({
        error: error.message,
        details: error.response?.data,
      });
    }
  }
});

/**
 * POST /api/verify-email
 * Verify email address
 */
app.post('/api/verify-email', async (req, res) => {
  try {
    const { email, token } = req.body;
    const result = await authClient.verifyEmail(email, token);
    res.json(result);
  } catch (error) {
    res.status(error.response?.status || 500).json({
      error: error.message,
      details: error.response?.data,
    });
  }
});

/**
 * Protected route example
 */
app.get('/api/protected', requireAuth, async (req, res) => {
  try {
    // Example of making an authenticated request to another service
    const response = await authClient.authenticatedRequest('GET', '/some-protected-endpoint');
    res.json(response.data);
  } catch (error) {
    res.status(error.response?.status || 500).json({
      error: error.message,
    });
  }
});

// Static files (for a simple frontend)
app.use(express.static('public'));

// Error handling
app.use((err, req, res, next) => {
  console.error(err.stack);
  res.status(500).json({ error: 'Internal server error' });
});

// Start server
const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Example app listening on port ${PORT}`);
  console.log(`Auth service URL: ${authClient.baseURL}`);
});

// Handle auth client events
authClient.on('tokenRefreshed', ({ success }) => {
  if (success) {
    console.log('Token automatically refreshed');
  }
});

authClient.on('autoRefreshFailed', ({ error }) => {
  console.error('Auto-refresh failed:', error.message);
});
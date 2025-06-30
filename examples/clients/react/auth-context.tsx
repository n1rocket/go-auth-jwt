import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';

interface User {
  id: string;
  email: string;
  emailVerified: boolean;
  createdAt: string;
}

interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

interface AuthContextType {
  user: User | null;
  loading: boolean;
  error: string | null;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<void>;
  verifyEmail: (email: string, token: string) => Promise<void>;
  clearError: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const API_BASE_URL = process.env.REACT_APP_AUTH_API_URL || 'http://localhost:8080/api/v1';

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tokens, setTokens] = useState<AuthTokens | null>(null);
  const [refreshTimer, setRefreshTimer] = useState<NodeJS.Timeout | null>(null);

  // Make authenticated request
  const authenticatedFetch = useCallback(async (url: string, options: RequestInit = {}) => {
    if (!tokens?.accessToken) {
      throw new Error('Not authenticated');
    }

    const response = await fetch(url, {
      ...options,
      headers: {
        ...options.headers,
        'Authorization': `Bearer ${tokens.accessToken}`,
        'Content-Type': 'application/json',
      },
    });

    if (response.status === 401) {
      // Token might be expired, try to refresh
      await refreshToken();
      // Retry with new token
      if (tokens?.accessToken) {
        return fetch(url, {
          ...options,
          headers: {
            ...options.headers,
            'Authorization': `Bearer ${tokens.accessToken}`,
            'Content-Type': 'application/json',
          },
        });
      }
    }

    return response;
  }, [tokens]);

  // Schedule token refresh
  const scheduleTokenRefresh = useCallback((expiresIn: number) => {
    // Clear existing timer
    if (refreshTimer) {
      clearTimeout(refreshTimer);
    }

    // Refresh 30 seconds before expiration
    const refreshTime = (expiresIn - 30) * 1000;
    if (refreshTime > 0) {
      const timer = setTimeout(() => {
        refreshToken().catch(console.error);
      }, refreshTime);
      setRefreshTimer(timer);
    }
  }, [refreshTimer]);

  // Load user profile
  const loadUserProfile = useCallback(async () => {
    try {
      const response = await authenticatedFetch(`${API_BASE_URL}/auth/me`);
      if (!response.ok) {
        throw new Error('Failed to load profile');
      }
      const userData = await response.json();
      setUser(userData);
    } catch (err) {
      console.error('Failed to load user profile:', err);
      setUser(null);
    }
  }, [authenticatedFetch]);

  // Initialize auth state from localStorage
  useEffect(() => {
    const storedTokens = localStorage.getItem('auth_tokens');
    if (storedTokens) {
      try {
        const parsedTokens = JSON.parse(storedTokens);
        setTokens(parsedTokens);
        loadUserProfile();
      } catch (err) {
        console.error('Failed to parse stored tokens:', err);
        localStorage.removeItem('auth_tokens');
      }
    }
    setLoading(false);
  }, []);

  // Save tokens to localStorage when they change
  useEffect(() => {
    if (tokens) {
      localStorage.setItem('auth_tokens', JSON.stringify(tokens));
      scheduleTokenRefresh(tokens.expiresIn);
    } else {
      localStorage.removeItem('auth_tokens');
      if (refreshTimer) {
        clearTimeout(refreshTimer);
      }
    }
  }, [tokens, scheduleTokenRefresh, refreshTimer]);

  const login = async (email: string, password: string) => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch(`${API_BASE_URL}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Login failed');
      }

      setTokens({
        accessToken: data.access_token,
        refreshToken: data.refresh_token,
        expiresIn: data.expires_in,
      });

      await loadUserProfile();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
      throw err;
    } finally {
      setLoading(false);
    }
  };

  const signup = async (email: string, password: string) => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE_URL}/auth/signup`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Signup failed');
      }

      // Auto-login after signup (optional)
      // await login(email, password);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Signup failed');
      throw err;
    } finally {
      setLoading(false);
    }
  };

  const logout = async () => {
    setLoading(true);
    
    try {
      if (tokens?.refreshToken) {
        await fetch(`${API_BASE_URL}/auth/logout`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${tokens.accessToken}`,
          },
          body: JSON.stringify({ refresh_token: tokens.refreshToken }),
        });
      }
    } catch (err) {
      console.error('Logout API call failed:', err);
    } finally {
      setUser(null);
      setTokens(null);
      setLoading(false);
    }
  };

  const refreshToken = async () => {
    if (!tokens?.refreshToken) {
      throw new Error('No refresh token available');
    }

    try {
      const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: tokens.refreshToken }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Token refresh failed');
      }

      setTokens({
        accessToken: data.access_token,
        refreshToken: data.refresh_token,
        expiresIn: data.expires_in,
      });
    } catch (err) {
      // Refresh failed, clear auth state
      setUser(null);
      setTokens(null);
      throw err;
    }
  };

  const verifyEmail = async (email: string, token: string) => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE_URL}/auth/verify-email`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, token }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || 'Email verification failed');
      }

      // Reload user profile if logged in
      if (user) {
        await loadUserProfile();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Email verification failed');
      throw err;
    } finally {
      setLoading(false);
    }
  };

  const clearError = () => setError(null);

  const value: AuthContextType = {
    user,
    loading,
    error,
    login,
    signup,
    logout,
    refreshToken,
    verifyEmail,
    clearError,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

// Higher-order component for protected routes
export const withAuth = <P extends object>(
  Component: React.ComponentType<P>
): React.FC<P> => {
  return (props: P) => {
    const { user, loading } = useAuth();

    if (loading) {
      return <div>Loading...</div>;
    }

    if (!user) {
      // Redirect to login or show unauthorized message
      return <div>Please login to access this page</div>;
    }

    return <Component {...props} />;
  };
};
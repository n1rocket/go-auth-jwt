"""
JWT Auth Service Client for Python

Example usage of the JWT authentication service from Python applications.
"""

import json
import time
import threading
from datetime import datetime, timedelta
from typing import Dict, Optional, Tuple, Any
from urllib.parse import urljoin
import requests
from requests.adapters import HTTPAdapter
from requests.packages.urllib3.util.retry import Retry


class JWTAuthClient:
    """Client for interacting with the JWT Authentication Service."""
    
    def __init__(self, base_url: str = "http://localhost:8080", 
                 api_path: str = "/api/v1",
                 auto_refresh: bool = True,
                 timeout: int = 30):
        """
        Initialize the JWT Auth Client.
        
        Args:
            base_url: Base URL of the auth service
            api_path: API path prefix
            auto_refresh: Enable automatic token refresh
            timeout: Request timeout in seconds
        """
        self.base_url = base_url.rstrip('/')
        self.api_path = api_path.rstrip('/')
        self.auto_refresh = auto_refresh
        self.timeout = timeout
        
        self.access_token: Optional[str] = None
        self.refresh_token: Optional[str] = None
        self.token_expires_at: Optional[datetime] = None
        self._refresh_timer: Optional[threading.Timer] = None
        
        # Setup session with retry strategy
        self.session = requests.Session()
        retry_strategy = Retry(
            total=3,
            backoff_factor=1,
            status_forcelist=[429, 500, 502, 503, 504],
        )
        adapter = HTTPAdapter(max_retries=retry_strategy)
        self.session.mount("http://", adapter)
        self.session.mount("https://", adapter)
    
    def _make_request(self, method: str, endpoint: str, 
                     data: Optional[Dict] = None,
                     authenticated: bool = False) -> Dict[str, Any]:
        """Make an HTTP request to the auth service."""
        url = urljoin(self.base_url, f"{self.api_path}{endpoint}")
        
        headers = {"Content-Type": "application/json"}
        if authenticated and self.access_token:
            headers["Authorization"] = f"Bearer {self.access_token}"
        
        try:
            response = self.session.request(
                method=method,
                url=url,
                json=data,
                headers=headers,
                timeout=self.timeout
            )
            response.raise_for_status()
            return response.json() if response.content else {}
            
        except requests.exceptions.HTTPError as e:
            if e.response.status_code == 401 and authenticated and self.refresh_token:
                # Try to refresh token and retry
                try:
                    self.refresh()
                    headers["Authorization"] = f"Bearer {self.access_token}"
                    response = self.session.request(
                        method=method,
                        url=url,
                        json=data,
                        headers=headers,
                        timeout=self.timeout
                    )
                    response.raise_for_status()
                    return response.json() if response.content else {}
                except Exception:
                    pass
            
            # Parse error response
            try:
                error_data = e.response.json()
                raise JWTAuthError(
                    message=error_data.get("message", str(e)),
                    status_code=e.response.status_code,
                    error_code=error_data.get("code"),
                    details=error_data.get("details")
                )
            except (json.JSONDecodeError, KeyError):
                raise JWTAuthError(
                    message=str(e),
                    status_code=e.response.status_code
                )
    
    def signup(self, email: str, password: str) -> Dict[str, Any]:
        """Register a new user."""
        return self._make_request(
            "POST", 
            "/auth/signup",
            data={"email": email, "password": password}
        )
    
    def login(self, email: str, password: str) -> Dict[str, Any]:
        """Login with email and password."""
        response = self._make_request(
            "POST",
            "/auth/login",
            data={"email": email, "password": password}
        )
        
        # Store tokens
        self.access_token = response["access_token"]
        self.refresh_token = response["refresh_token"]
        expires_in = response.get("expires_in", 3600)
        self.token_expires_at = datetime.now() + timedelta(seconds=expires_in)
        
        # Schedule token refresh
        if self.auto_refresh:
            self._schedule_token_refresh(expires_in)
        
        return response
    
    def refresh(self) -> Dict[str, Any]:
        """Refresh the access token."""
        if not self.refresh_token:
            raise JWTAuthError("No refresh token available")
        
        response = self._make_request(
            "POST",
            "/auth/refresh",
            data={"refresh_token": self.refresh_token}
        )
        
        # Update tokens
        self.access_token = response["access_token"]
        self.refresh_token = response["refresh_token"]
        expires_in = response.get("expires_in", 3600)
        self.token_expires_at = datetime.now() + timedelta(seconds=expires_in)
        
        # Reschedule token refresh
        if self.auto_refresh:
            self._schedule_token_refresh(expires_in)
        
        return response
    
    def logout(self) -> Dict[str, Any]:
        """Logout and revoke the refresh token."""
        if not self.refresh_token:
            return {"message": "Already logged out"}
        
        try:
            response = self._make_request(
                "POST",
                "/auth/logout",
                data={"refresh_token": self.refresh_token},
                authenticated=True
            )
        finally:
            # Clear tokens regardless of response
            self._clear_tokens()
        
        return response
    
    def logout_all(self) -> Dict[str, Any]:
        """Logout from all devices."""
        try:
            response = self._make_request(
                "POST",
                "/auth/logout-all",
                authenticated=True
            )
        finally:
            self._clear_tokens()
        
        return response
    
    def get_profile(self) -> Dict[str, Any]:
        """Get current user profile."""
        return self._make_request("GET", "/auth/me", authenticated=True)
    
    def verify_email(self, email: str, token: str) -> Dict[str, Any]:
        """Verify email address with token."""
        return self._make_request(
            "POST",
            "/auth/verify-email",
            data={"email": email, "token": token}
        )
    
    def resend_verification(self) -> Dict[str, Any]:
        """Resend verification email."""
        return self._make_request(
            "POST",
            "/auth/resend-verification",
            authenticated=True
        )
    
    def authenticated_request(self, method: str, endpoint: str,
                            data: Optional[Dict] = None) -> Dict[str, Any]:
        """Make an authenticated request to any endpoint."""
        return self._make_request(method, endpoint, data, authenticated=True)
    
    def is_authenticated(self) -> bool:
        """Check if user is authenticated."""
        return bool(self.access_token)
    
    def get_tokens(self) -> Tuple[Optional[str], Optional[str]]:
        """Get current tokens (for persistence)."""
        return self.access_token, self.refresh_token
    
    def set_tokens(self, access_token: str, refresh_token: str,
                   expires_in: int = 3600) -> None:
        """Set tokens (for restoration from storage)."""
        self.access_token = access_token
        self.refresh_token = refresh_token
        self.token_expires_at = datetime.now() + timedelta(seconds=expires_in)
        
        if self.auto_refresh:
            self._schedule_token_refresh(expires_in)
    
    def _schedule_token_refresh(self, expires_in: int) -> None:
        """Schedule automatic token refresh."""
        self._cancel_refresh_timer()
        
        # Refresh 30 seconds before expiration
        refresh_time = expires_in - 30
        if refresh_time > 0:
            self._refresh_timer = threading.Timer(
                refresh_time,
                self._auto_refresh_token
            )
            self._refresh_timer.daemon = True
            self._refresh_timer.start()
    
    def _auto_refresh_token(self) -> None:
        """Automatically refresh the token."""
        try:
            self.refresh()
            print("Token automatically refreshed")
        except Exception as e:
            print(f"Auto-refresh failed: {e}")
    
    def _cancel_refresh_timer(self) -> None:
        """Cancel the refresh timer."""
        if self._refresh_timer:
            self._refresh_timer.cancel()
            self._refresh_timer = None
    
    def _clear_tokens(self) -> None:
        """Clear all tokens and cancel refresh timer."""
        self.access_token = None
        self.refresh_token = None
        self.token_expires_at = None
        self._cancel_refresh_timer()
    
    def close(self) -> None:
        """Close the client and cleanup resources."""
        self._cancel_refresh_timer()
        self.session.close()
    
    def __enter__(self):
        """Context manager entry."""
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.close()


class JWTAuthError(Exception):
    """Exception raised by JWT Auth Client."""
    
    def __init__(self, message: str, status_code: int = None,
                 error_code: str = None, details: Dict = None):
        super().__init__(message)
        self.message = message
        self.status_code = status_code
        self.error_code = error_code
        self.details = details


# Example usage
if __name__ == "__main__":
    import asyncio
    import os
    
    # Get configuration from environment
    auth_url = os.getenv("AUTH_SERVICE_URL", "http://localhost:8080")
    
    # Example synchronous usage
    def example_sync():
        with JWTAuthClient(base_url=auth_url) as client:
            try:
                # Signup
                print("1. Signing up new user...")
                result = client.signup("test@example.com", "SecurePassword123!")
                print(f"   ✓ Signup successful: {result}")
                
                # Login
                print("\n2. Logging in...")
                result = client.login("test@example.com", "SecurePassword123!")
                print(f"   ✓ Login successful")
                print(f"   Access token: {result['access_token'][:20]}...")
                
                # Get profile
                print("\n3. Getting user profile...")
                profile = client.get_profile()
                print(f"   ✓ Profile: {profile}")
                
                # Make authenticated request
                print("\n4. Making authenticated request...")
                # result = client.authenticated_request("GET", "/some-endpoint")
                
                # Wait a bit to see auto-refresh
                print("\n5. Waiting for auto-refresh...")
                time.sleep(5)
                
                # Logout
                print("\n6. Logging out...")
                result = client.logout()
                print(f"   ✓ Logout successful: {result}")
                
            except JWTAuthError as e:
                print(f"Auth error: {e.message}")
                if e.details:
                    print(f"Details: {e.details}")
            except Exception as e:
                print(f"Error: {e}")
    
    # Run example
    example_sync()
"""
Example Flask application using JWT Auth Client
"""

import os
from functools import wraps
from flask import Flask, request, jsonify, session, g
from jwt_auth_client import JWTAuthClient, JWTAuthError

app = Flask(__name__)
app.secret_key = os.getenv('SECRET_KEY', 'your-secret-key-here')

# Initialize auth client
auth_client = JWTAuthClient(
    base_url=os.getenv('AUTH_SERVICE_URL', 'http://localhost:8080'),
    auto_refresh=True
)


def login_required(f):
    """Decorator to require authentication."""
    @wraps(f)
    def decorated_function(*args, **kwargs):
        if 'tokens' not in session:
            return jsonify({'error': 'Authentication required'}), 401
        
        # Restore tokens to client
        try:
            auth_client.set_tokens(
                session['tokens']['access_token'],
                session['tokens']['refresh_token']
            )
            g.user = None  # Could fetch user profile here if needed
        except Exception:
            session.pop('tokens', None)
            return jsonify({'error': 'Invalid session'}), 401
        
        return f(*args, **kwargs)
    
    return decorated_function


@app.route('/api/signup', methods=['POST'])
def signup():
    """Register a new user."""
    try:
        data = request.get_json()
        result = auth_client.signup(data['email'], data['password'])
        return jsonify(result), 201
    except JWTAuthError as e:
        return jsonify({
            'error': e.message,
            'code': e.error_code,
            'details': e.details
        }), e.status_code or 400
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/login', methods=['POST'])
def login():
    """Login user."""
    try:
        data = request.get_json()
        result = auth_client.login(data['email'], data['password'])
        
        # Store tokens in session
        session['tokens'] = {
            'access_token': result['access_token'],
            'refresh_token': result['refresh_token']
        }
        
        return jsonify({
            'message': 'Login successful',
            'expires_in': result.get('expires_in', 3600)
        }), 200
    except JWTAuthError as e:
        return jsonify({
            'error': e.message,
            'code': e.error_code
        }), e.status_code or 401
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/logout', methods=['POST'])
@login_required
def logout():
    """Logout user."""
    try:
        auth_client.logout()
    except Exception:
        pass  # Continue with local logout even if API fails
    
    session.pop('tokens', None)
    return jsonify({'message': 'Logout successful'}), 200


@app.route('/api/profile')
@login_required
def get_profile():
    """Get user profile."""
    try:
        profile = auth_client.get_profile()
        return jsonify(profile), 200
    except JWTAuthError as e:
        if e.status_code == 401:
            # Try to refresh token
            try:
                auth_client.refresh()
                # Update session with new tokens
                access_token, refresh_token = auth_client.get_tokens()
                session['tokens'] = {
                    'access_token': access_token,
                    'refresh_token': refresh_token
                }
                # Retry request
                profile = auth_client.get_profile()
                return jsonify(profile), 200
            except Exception:
                session.pop('tokens', None)
                return jsonify({'error': 'Session expired'}), 401
        
        return jsonify({'error': e.message}), e.status_code or 500
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/verify-email', methods=['POST'])
def verify_email():
    """Verify email address."""
    try:
        data = request.get_json()
        result = auth_client.verify_email(data['email'], data['token'])
        return jsonify(result), 200
    except JWTAuthError as e:
        return jsonify({'error': e.message}), e.status_code or 400
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/api/protected')
@login_required
def protected_route():
    """Example protected route."""
    try:
        # You can make authenticated requests to other services here
        # result = auth_client.authenticated_request('GET', '/some-endpoint')
        
        return jsonify({
            'message': 'This is a protected route',
            'user': g.user if hasattr(g, 'user') else None
        }), 200
    except Exception as e:
        return jsonify({'error': str(e)}), 500


@app.route('/health')
def health():
    """Health check endpoint."""
    return jsonify({'status': 'healthy'}), 200


@app.errorhandler(404)
def not_found(error):
    return jsonify({'error': 'Not found'}), 404


@app.errorhandler(500)
def internal_error(error):
    return jsonify({'error': 'Internal server error'}), 500


# Middleware to add user to context
@app.before_request
def load_user():
    """Load user for authenticated requests."""
    if 'tokens' in session:
        try:
            auth_client.set_tokens(
                session['tokens']['access_token'],
                session['tokens']['refresh_token']
            )
            # Could fetch and cache user profile here
            # g.user = auth_client.get_profile()
        except Exception:
            pass


# Example WebSocket support (using Flask-SocketIO)
try:
    from flask_socketio import SocketIO, emit, disconnect
    
    socketio = SocketIO(app, cors_allowed_origins="*")
    
    @socketio.on('authenticate')
    def handle_authenticate(data):
        """Authenticate WebSocket connection."""
        try:
            # Validate token
            auth_client.set_tokens(data['token'], '')
            profile = auth_client.get_profile()
            
            # Store in session
            session['ws_authenticated'] = True
            session['ws_user'] = profile
            
            emit('authenticated', {'user': profile})
        except Exception as e:
            emit('error', {'message': 'Authentication failed'})
            disconnect()
    
    @socketio.on('message')
    def handle_message(data):
        """Handle authenticated WebSocket message."""
        if not session.get('ws_authenticated'):
            emit('error', {'message': 'Not authenticated'})
            return
        
        # Handle message
        emit('response', {
            'message': f"Echo: {data['message']}",
            'user': session.get('ws_user')
        })
except ImportError:
    socketio = None


if __name__ == '__main__':
    port = int(os.getenv('PORT', 5000))
    debug = os.getenv('FLASK_ENV') == 'development'
    
    print(f"Starting Flask app on port {port}")
    print(f"Auth service URL: {auth_client.base_url}")
    
    if socketio:
        socketio.run(app, host='0.0.0.0', port=port, debug=debug)
    else:
        app.run(host='0.0.0.0', port=port, debug=debug)
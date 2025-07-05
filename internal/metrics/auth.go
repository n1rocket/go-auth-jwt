package metrics

// AuthMetrics contains all authentication-related metrics
type AuthMetrics struct {
	LoginAttempts   *Counter
	LoginSuccess    *Counter
	LoginFailure    *Counter
	SignupAttempts  *Counter
	SignupSuccess   *Counter
	SignupFailure   *Counter
	TokensIssued    *Counter
	TokensRefreshed *Counter
	TokensRevoked   *Counter
	ActiveSessions  *Gauge
}

// NewAuthMetrics creates a new AuthMetrics instance
func NewAuthMetrics() *AuthMetrics {
	return &AuthMetrics{
		LoginAttempts:   NewCounter("auth_login_attempts_total", "Total number of login attempts"),
		LoginSuccess:    NewCounter("auth_login_success_total", "Total number of successful logins"),
		LoginFailure:    NewCounter("auth_login_failure_total", "Total number of failed logins"),
		SignupAttempts:  NewCounter("auth_signup_attempts_total", "Total number of signup attempts"),
		SignupSuccess:   NewCounter("auth_signup_success_total", "Total number of successful signups"),
		SignupFailure:   NewCounter("auth_signup_failure_total", "Total number of failed signups"),
		TokensIssued:    NewCounter("auth_tokens_issued_total", "Total number of tokens issued"),
		TokensRefreshed: NewCounter("auth_tokens_refreshed_total", "Total number of tokens refreshed"),
		TokensRevoked:   NewCounter("auth_tokens_revoked_total", "Total number of tokens revoked"),
		ActiveSessions:  NewGauge("auth_active_sessions", "Number of active user sessions"),
	}
}

// Register registers all auth metrics
func (a *AuthMetrics) Register(registry MetricRegistry) {
	registry.Register(a.LoginAttempts)
	registry.Register(a.LoginSuccess)
	registry.Register(a.LoginFailure)
	registry.Register(a.SignupAttempts)
	registry.Register(a.SignupSuccess)
	registry.Register(a.SignupFailure)
	registry.Register(a.TokensIssued)
	registry.Register(a.TokensRefreshed)
	registry.Register(a.TokensRevoked)
	registry.Register(a.ActiveSessions)
}

// RecordLogin records a login attempt
func (a *AuthMetrics) RecordLogin(success bool) {
	a.LoginAttempts.Inc()
	if success {
		a.LoginSuccess.Inc()
		a.ActiveSessions.Inc()
	} else {
		a.LoginFailure.Inc()
	}
}

// RecordSignup records a signup attempt
func (a *AuthMetrics) RecordSignup(success bool) {
	a.SignupAttempts.Inc()
	if success {
		a.SignupSuccess.Inc()
	} else {
		a.SignupFailure.Inc()
	}
}

// RecordTokenIssued records a token issuance
func (a *AuthMetrics) RecordTokenIssued() {
	a.TokensIssued.Inc()
}

// RecordTokenRefreshed records a token refresh
func (a *AuthMetrics) RecordTokenRefreshed() {
	a.TokensRefreshed.Inc()
}

// RecordTokenRevoked records a token revocation
func (a *AuthMetrics) RecordTokenRevoked() {
	a.TokensRevoked.Inc()
}

// RecordLogout records a logout
func (a *AuthMetrics) RecordLogout() {
	a.ActiveSessions.Dec()
}
package metrics

// BusinessMetrics contains all business-related metrics
type BusinessMetrics struct {
	UsersTotal        *Counter
	UsersActive       *Gauge
	UsersVerified     *Counter
	PasswordResets    *Counter
	VerificationsSent *Counter
}

// NewBusinessMetrics creates a new BusinessMetrics instance
func NewBusinessMetrics() *BusinessMetrics {
	return &BusinessMetrics{
		UsersTotal:        NewCounter("users_total", "Total number of registered users"),
		UsersActive:       NewGauge("users_active", "Number of active users"),
		UsersVerified:     NewCounter("users_verified_total", "Total number of verified users"),
		PasswordResets:    NewCounter("password_resets_total", "Total number of password reset requests"),
		VerificationsSent: NewCounter("verifications_sent_total", "Total number of verification emails sent"),
	}
}

// Register registers all business metrics
func (b *BusinessMetrics) Register(registry MetricRegistry) {
	registry.Register(b.UsersTotal)
	registry.Register(b.UsersActive)
	registry.Register(b.UsersVerified)
	registry.Register(b.PasswordResets)
	registry.Register(b.VerificationsSent)
}

// RecordUserRegistered records a new user registration
func (b *BusinessMetrics) RecordUserRegistered() {
	b.UsersTotal.Inc()
}

// RecordUserVerified records a user verification
func (b *BusinessMetrics) RecordUserVerified() {
	b.UsersVerified.Inc()
}

// RecordPasswordReset records a password reset request
func (b *BusinessMetrics) RecordPasswordReset() {
	b.PasswordResets.Inc()
}

// RecordVerificationSent records a verification email sent
func (b *BusinessMetrics) RecordVerificationSent() {
	b.VerificationsSent.Inc()
}

// SetActiveUsers sets the number of active users
func (b *BusinessMetrics) SetActiveUsers(count float64) {
	b.UsersActive.Set(count)
}
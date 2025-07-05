package context

// ContextKey is a type for context keys
type ContextKey string

// Context keys for user information
const (
	UserIDKey            ContextKey = "user_id"
	UserEmailKey         ContextKey = "user_email"
	UserEmailVerifiedKey ContextKey = "user_email_verified"
)

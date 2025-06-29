package security

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNewPasswordHasher(t *testing.T) {
	tests := []struct {
		name     string
		cost     int
		wantCost int
	}{
		{"default cost", DefaultCost, DefaultCost},
		{"minimum cost", MinCost, MinCost},
		{"maximum cost", MaxCost, MaxCost},
		{"below minimum", MinCost - 5, MinCost},
		{"above maximum", MaxCost + 5, MaxCost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ph := NewPasswordHasher(tt.cost)
			if ph.cost != tt.wantCost {
				t.Errorf("NewPasswordHasher() cost = %v, want %v", ph.cost, tt.wantCost)
			}
		})
	}
}

func TestPasswordHasher_Hash(t *testing.T) {
	ph := NewDefaultPasswordHasher()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"simple password", "password123", false},
		{"complex password", "P@ssw0rd!123#Complex", false},
		{"empty password", "", false}, // bcrypt accepts empty passwords
		{"unicode password", "пароль密码🔐", false},
		{"very long password", strings.Repeat("a", 72), false},  // bcrypt has 72 byte limit
		{"password exceeds limit", strings.Repeat("a", 73), true}, // Should fail
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := ph.Hash(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("Hash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify it's a valid bcrypt hash
				if !strings.HasPrefix(hash, "$2a$") {
					t.Errorf("Hash() returned invalid bcrypt hash format")
				}

				// Verify the hash is different from the password
				if hash == tt.password {
					t.Errorf("Hash() returned unhashed password")
				}

				// Verify we can compare it successfully
				if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(tt.password)); err != nil {
					t.Errorf("Generated hash doesn't match password: %v", err)
				}
			}
		})
	}
}

func TestPasswordHasher_Compare(t *testing.T) {
	ph := NewDefaultPasswordHasher()

	// Generate a test hash
	password := "testPassword123"
	validHash, _ := ph.Hash(password)

	tests := []struct {
		name     string
		password string
		hash     string
		wantErr  bool
	}{
		{"valid password", password, validHash, false},
		{"wrong password", "wrongPassword", validHash, true},
		{"empty password", "", validHash, true},
		{"invalid hash", password, "invalid-hash", true},
		{"empty hash", password, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ph.Compare(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"default length", 0},
		{"custom length 16", 16},
		{"custom length 32", 32},
		{"custom length 64", 64},
		{"negative length", -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token1, err := GenerateToken(tt.length)
			if err != nil {
				t.Errorf("GenerateToken() error = %v", err)
				return
			}

			// Check token is not empty
			if token1 == "" {
				t.Error("GenerateToken() returned empty token")
			}

			// Generate another token to ensure uniqueness
			token2, err := GenerateToken(tt.length)
			if err != nil {
				t.Errorf("GenerateToken() error = %v", err)
				return
			}

			// Tokens should be different
			if token1 == token2 {
				t.Error("GenerateToken() returned duplicate tokens")
			}

			// Check that token is URL-safe (no padding)
			if strings.Contains(token1, "=") {
				t.Error("GenerateToken() returned token with padding")
			}
		})
	}
}

func TestGenerateSecureToken(t *testing.T) {
	tests := []struct {
		name       string
		byteLength int
	}{
		{"default length", 0},
		{"16 bytes", 16},
		{"32 bytes", 32},
		{"64 bytes", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token1, err := GenerateSecureToken(tt.byteLength)
			if err != nil {
				t.Errorf("GenerateSecureToken() error = %v", err)
				return
			}

			// Check token is not empty
			if token1 == "" {
				t.Error("GenerateSecureToken() returned empty token")
			}

			// Generate another token to ensure uniqueness
			token2, err := GenerateSecureToken(tt.byteLength)
			if err != nil {
				t.Errorf("GenerateSecureToken() error = %v", err)
				return
			}

			// Tokens should be different
			if token1 == token2 {
				t.Error("GenerateSecureToken() returned duplicate tokens")
			}
		})
	}
}

func TestConstantTimeCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"equal strings", "token123", "token123", true},
		{"different strings", "token123", "token456", false},
		{"different lengths", "short", "longer-string", false},
		{"empty strings", "", "", true},
		{"one empty", "token", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConstantTimeCompare(tt.a, tt.b); got != tt.want {
				t.Errorf("ConstantTimeCompare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name    string
		password string
		wantErr bool
	}{
		{"valid simple password", "password123", false},
		{"exactly 8 chars", "12345678", false},
		{"complex password", "P@ssw0rd!123", false},
		{"too short", "1234567", true},
		{"empty password", "", true},
		{"unicode password", "пароль12", false},
		{"very long password", strings.Repeat("a", 72), false},
		// If you enable stronger requirements, update these tests
		// {"no uppercase", "password123!", true},
		// {"no lowercase", "PASSWORD123!", true},
		// {"no number", "Password!", true},
		// {"no special", "Password123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordStrength() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkPasswordHasher_Hash(b *testing.B) {
	ph := NewDefaultPasswordHasher()
	password := "BenchmarkPassword123!"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ph.Hash(password)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPasswordHasher_Compare(b *testing.B) {
	ph := NewDefaultPasswordHasher()
	password := "BenchmarkPassword123!"
	hash, _ := ph.Hash(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ph.Compare(password, hash)
		if err != nil {
			b.Fatal(err)
		}
	}
}
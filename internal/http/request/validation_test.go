package request

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name    string
		body    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid JSON",
			body:    `{"name":"test","value":123}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			body:    `{"name":"test","value":}`,
			wantErr: true,
			errMsg:  "failed to decode JSON",
		},
		{
			name:    "unknown fields",
			body:    `{"name":"test","value":123,"unknown":"field"}`,
			wantErr: true,
			errMsg:  "failed to decode JSON",
		},
		{
			name:    "multiple JSON objects",
			body:    `{"name":"test","value":123}{"name":"test2","value":456}`,
			wantErr: true,
			errMsg:  "must contain only one JSON object",
		},
		{
			name:    "empty body",
			body:    ``,
			wantErr: true,
			errMsg:  "failed to decode JSON",
		},
		{
			name:    "body too large",
			body:    strings.Repeat("a", MaxRequestBodySize+1),
			wantErr: true,
			errMsg:  "failed to decode JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))

			var dst testStruct
			err := DecodeJSON(req, &dst)

			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidateContentType(t *testing.T) {
	tests := []struct {
		name         string
		contentType  string
		expectedType string
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "valid content type",
			contentType:  "application/json",
			expectedType: "application/json",
			wantErr:      false,
		},
		{
			name:         "valid with charset",
			contentType:  "application/json; charset=utf-8",
			expectedType: "application/json",
			wantErr:      false,
		},
		{
			name:         "invalid content type",
			contentType:  "text/plain",
			expectedType: "application/json",
			wantErr:      true,
			errMsg:       "Content-Type must be application/json",
		},
		{
			name:         "missing content type",
			contentType:  "",
			expectedType: "application/json",
			wantErr:      true,
			errMsg:       "Content-Type header is required",
		},
		{
			name:         "content type with spaces",
			contentType:  " application/json ",
			expectedType: "application/json",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			err := ValidateContentType(req, tt.expectedType)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContentType() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestValidateJSONRequest(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name        string
		contentType string
		body        string
		wantErr     bool
	}{
		{
			name:        "valid request",
			contentType: "application/json",
			body:        `{"name":"test"}`,
			wantErr:     false,
		},
		{
			name:        "wrong content type",
			contentType: "text/plain",
			body:        `{"name":"test"}`,
			wantErr:     true,
		},
		{
			name:        "invalid JSON",
			contentType: "application/json",
			body:        `{"name":}`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			var dst testStruct
			err := ValidateJSONRequest(req, &dst)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJSONRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		fields map[string]string
		want   int // number of validation errors
	}{
		{
			name: "all fields present",
			fields: map[string]string{
				"email":    "test@example.com",
				"password": "password123",
			},
			want: 0,
		},
		{
			name: "empty field",
			fields: map[string]string{
				"email":    "test@example.com",
				"password": "",
			},
			want: 1,
		},
		{
			name: "whitespace only",
			fields: map[string]string{
				"email":    "   ",
				"password": "\t\n",
			},
			want: 2,
		},
		{
			name:   "empty map",
			fields: map[string]string{},
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateRequiredFields(tt.fields)
			if len(got) != tt.want {
				t.Errorf("ValidateRequiredFields() returned %d errors, want %d", len(got), tt.want)
			}

			// Check error structure
			for _, err := range got {
				if err.Code != "REQUIRED_FIELD" {
					t.Errorf("Expected error code REQUIRED_FIELD, got %s", err.Code)
				}
				if err.Field == "" {
					t.Error("Error field should not be empty")
				}
				if err.Message == "" {
					t.Error("Error message should not be empty")
				}
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "valid with subdomain",
			email:   "test@mail.example.com",
			wantErr: false,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			email:   "   ",
			wantErr: true,
		},
		{
			name:    "missing @",
			email:   "testexample.com",
			wantErr: true,
		},
		{
			name:    "multiple @",
			email:   "test@@example.com",
			wantErr: true,
		},
		{
			name:    "missing domain",
			email:   "test@",
			wantErr: true,
		},
		{
			name:    "missing local part",
			email:   "@example.com",
			wantErr: true,
		},
		{
			name:    "missing dot in domain",
			email:   "test@example",
			wantErr: true,
		},
		{
			name:    "with spaces (trimmed)",
			email:   "  test@example.com  ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "minimum length",
			password: "12345678",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "1234567",
			wantErr:  true,
			errMsg:   "at least 8 characters",
		},
		{
			name:     "too long",
			password: strings.Repeat("a", 73),
			wantErr:  true,
			errMsg:   "not exceed 72 characters",
		},
		{
			name:     "maximum length",
			password: strings.Repeat("a", 72),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		maxLength int
		want      string
	}{
		{
			name:      "normal string",
			input:     "hello world",
			maxLength: 20,
			want:      "hello world",
		},
		{
			name:      "with whitespace",
			input:     "  hello world  ",
			maxLength: 20,
			want:      "hello world",
		},
		{
			name:      "exceeds max length",
			input:     "hello world this is a long string",
			maxLength: 11,
			want:      "hello world",
		},
		{
			name:      "empty string",
			input:     "",
			maxLength: 10,
			want:      "",
		},
		{
			name:      "whitespace only",
			input:     "   \t\n   ",
			maxLength: 10,
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeString(tt.input, tt.maxLength)
			if got != tt.want {
				t.Errorf("SanitizeString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			token:   "   ",
			wantErr: true,
		},
		{
			name:    "too short",
			token:   "abc",
			wantErr: true,
		},
		{
			name:    "too long",
			token:   strings.Repeat("a", 1001),
			wantErr: true,
		},
		{
			name:    "minimum length",
			token:   "1234567890",
			wantErr: false,
		},
		{
			name:    "maximum length",
			token:   strings.Repeat("a", 1000),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		want       string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			want:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantErr:    false,
		},
		{
			name:       "missing header",
			authHeader: "",
			wantErr:    true,
			errMsg:     "Authorization header is required",
		},
		{
			name:       "wrong scheme",
			authHeader: "Basic dXNlcjpwYXNz",
			wantErr:    true,
			errMsg:     "must use Bearer scheme",
		},
		{
			name:       "missing token",
			authHeader: "Bearer ",
			wantErr:    true,
			errMsg:     "token is required",
		},
		{
			name:       "invalid format",
			authHeader: "Bearer",
			wantErr:    true,
			errMsg:     "must use Bearer scheme",
		},
		{
			name:       "multiple spaces",
			authHeader: "Bearer  token  extra",
			wantErr:    true,
			errMsg:     "must use Bearer scheme",
		},
		{
			name:       "token too short",
			authHeader: "Bearer abc",
			wantErr:    true,
			errMsg:     "invalid token format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			got, err := ExtractBearerToken(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractBearerToken() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("ExtractBearerToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test that DecodeJSON properly limits request body size
func TestDecodeJSON_BodySizeLimit(t *testing.T) {
	// Create a reader that simulates a large body
	largeBody := &infiniteReader{}
	req := httptest.NewRequest(http.MethodPost, "/", largeBody)

	var dst interface{}
	err := DecodeJSON(req, &dst)

	if err == nil {
		t.Error("Expected error for oversized body")
	}
}

// infiniteReader simulates an infinitely large request body
type infiniteReader struct {
	read int
}

func (r *infiniteReader) Read(p []byte) (n int, err error) {
	if r.read > MaxRequestBodySize {
		return 0, io.EOF
	}
	n = len(p)
	r.read += n
	for i := range p {
		p[i] = 'a'
	}
	return n, nil
}

// Test validation error type
func TestValidationError_Structure(t *testing.T) {
	errors := ValidateRequiredFields(map[string]string{
		"email": "",
	})

	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}

	err := errors[0]
	if err.Field != "email" {
		t.Errorf("Expected field 'email', got %q", err.Field)
	}
	if err.Code != "REQUIRED_FIELD" {
		t.Errorf("Expected code 'REQUIRED_FIELD', got %q", err.Code)
	}
	if !strings.Contains(err.Message, "required") {
		t.Errorf("Expected message to contain 'required', got %q", err.Message)
	}
}

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/n1rocket/go-auth-jwt/internal/domain"
)

func TestNewRefreshTokenRepository(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creating mock database: %v", err)
	}
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	
	if repo == nil {
		t.Error("Expected repository to be created")
	}
	
	if repo.db != db {
		t.Error("Expected db to be set correctly")
	}
}

func TestRefreshTokenRepository_Create(t *testing.T) {
	fixedTime := time.Now()
	
	tests := []struct {
		name    string
		token   *domain.RefreshToken
		setupMock func(sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "successful creation",
			token: &domain.RefreshToken{
				UserID:     "user-123",
				ExpiresAt:  fixedTime.Add(24 * time.Hour),
				Revoked:    false,
				CreatedAt:  fixedTime,
				LastUsedAt: fixedTime,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"token"}).
					AddRow("generated-token-uuid")
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO refresh_tokens`)).
					WithArgs(
						"user-123",
						fixedTime.Add(24 * time.Hour),
						false,
						nil,
						nil,
						nil,
						fixedTime,
						fixedTime,
					).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "with user agent and IP",
			token: &domain.RefreshToken{
				UserID:     "user-123",
				ExpiresAt:  fixedTime.Add(24 * time.Hour),
				Revoked:    false,
				UserAgent:  stringPtr("Mozilla/5.0"),
				IPAddress:  stringPtr("192.168.1.1"),
				CreatedAt:  fixedTime,
				LastUsedAt: fixedTime,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"token"}).
					AddRow("generated-token-uuid")
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO refresh_tokens`)).
					WithArgs(
						"user-123",
						fixedTime.Add(24 * time.Hour),
						false,
						nil,
						"Mozilla/5.0",
						"192.168.1.1",
						fixedTime,
						fixedTime,
					).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "database error",
			token: &domain.RefreshToken{
				UserID:     "user-123",
				ExpiresAt:  fixedTime.Add(24 * time.Hour),
				Revoked:    false,
				CreatedAt:  fixedTime,
				LastUsedAt: fixedTime,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO refresh_tokens`)).
					WithArgs(
						"user-123",
						fixedTime.Add(24 * time.Hour),
						false,
						nil,
						nil,
						nil,
						fixedTime,
						fixedTime,
					).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock database: %v", err)
			}
			defer db.Close()

			tt.setupMock(mock)
			
			repo := &RefreshTokenRepository{db: db}
			err = repo.Create(context.Background(), tt.token)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.token.Token == "" {
				t.Error("Expected token to be set")
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRefreshTokenRepository_GetByToken(t *testing.T) {
	fixedTime := time.Now()
	revokedTime := time.Now().Add(-1 * time.Hour)
	
	tests := []struct {
		name       string
		tokenValue string
		setupMock  func(sqlmock.Sqlmock)
		want       *domain.RefreshToken
		wantErr    bool
		errType    error
	}{
		{
			name:       "successful retrieval",
			tokenValue: "valid-token",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"token", "user_id", "expires_at", "revoked", "revoked_at",
					"user_agent", "ip_address", "created_at", "last_used_at",
				}).AddRow(
					"valid-token", "user-123", fixedTime.Add(24*time.Hour), false, nil,
					"Mozilla/5.0", "192.168.1.1", fixedTime, fixedTime,
				)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT token, user_id, expires_at`)).
					WithArgs("valid-token").
					WillReturnRows(rows)
			},
			want: &domain.RefreshToken{
				Token:      "valid-token",
				UserID:     "user-123",
				ExpiresAt:  fixedTime.Add(24 * time.Hour),
				Revoked:    false,
				UserAgent:  stringPtr("Mozilla/5.0"),
				IPAddress:  stringPtr("192.168.1.1"),
				CreatedAt:  fixedTime,
				LastUsedAt: fixedTime,
			},
			wantErr: false,
		},
		{
			name:       "revoked token",
			tokenValue: "revoked-token",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"token", "user_id", "expires_at", "revoked", "revoked_at",
					"user_agent", "ip_address", "created_at", "last_used_at",
				}).AddRow(
					"revoked-token", "user-123", fixedTime.Add(24*time.Hour), true, revokedTime,
					nil, nil, fixedTime, fixedTime,
				)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT token, user_id, expires_at`)).
					WithArgs("revoked-token").
					WillReturnRows(rows)
			},
			want: &domain.RefreshToken{
				Token:      "revoked-token",
				UserID:     "user-123",
				ExpiresAt:  fixedTime.Add(24 * time.Hour),
				Revoked:    true,
				RevokedAt:  &revokedTime,
				CreatedAt:  fixedTime,
				LastUsedAt: fixedTime,
			},
			wantErr: false,
		},
		{
			name:       "token not found",
			tokenValue: "non-existent",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT token, user_id, expires_at`)).
					WithArgs("non-existent").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errType: domain.ErrInvalidToken,
		},
		{
			name:       "database error",
			tokenValue: "error-token",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT token, user_id, expires_at`)).
					WithArgs("error-token").
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock database: %v", err)
			}
			defer db.Close()

			tt.setupMock(mock)
			
			repo := &RefreshTokenRepository{db: db}
			got, err := repo.GetByToken(context.Background(), tt.tokenValue)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("GetByToken() error = %v, want %v", err, tt.errType)
			}
			
			if !tt.wantErr && got != nil {
				if got.Token != tt.want.Token || got.UserID != tt.want.UserID {
					t.Errorf("GetByToken() = %v, want %v", got, tt.want)
				}
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRefreshTokenRepository_GetByUserID(t *testing.T) {
	fixedTime := time.Now()
	
	tests := []struct {
		name      string
		userID    string
		setupMock func(sqlmock.Sqlmock)
		want      int // number of tokens expected
		wantErr   bool
	}{
		{
			name:   "successful retrieval with multiple tokens",
			userID: "user-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"token", "user_id", "expires_at", "revoked", "revoked_at",
					"user_agent", "ip_address", "created_at", "last_used_at",
				}).
					AddRow("token-1", "user-123", fixedTime.Add(24*time.Hour), false, nil, nil, nil, fixedTime, fixedTime).
					AddRow("token-2", "user-123", fixedTime.Add(48*time.Hour), false, nil, nil, nil, fixedTime.Add(-1*time.Hour), fixedTime.Add(-1*time.Hour))
				
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT token, user_id, expires_at`)).
					WithArgs("user-123").
					WillReturnRows(rows)
			},
			want:    2,
			wantErr: false,
		},
		{
			name:   "no tokens for user",
			userID: "user-456",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"token", "user_id", "expires_at", "revoked", "revoked_at",
					"user_agent", "ip_address", "created_at", "last_used_at",
				})
				
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT token, user_id, expires_at`)).
					WithArgs("user-456").
					WillReturnRows(rows)
			},
			want:    0,
			wantErr: false,
		},
		{
			name:   "database error",
			userID: "user-789",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT token, user_id, expires_at`)).
					WithArgs("user-789").
					WillReturnError(errors.New("database error"))
			},
			want:    0,
			wantErr: true,
		},
		{
			name:   "scan error",
			userID: "user-scan",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"token", "user_id", "expires_at", "revoked", "revoked_at",
					"user_agent", "ip_address", "created_at", "last_used_at",
				}).
					AddRow("token-1", "user-scan", "invalid-time", false, nil, nil, nil, fixedTime, fixedTime) // invalid time will cause scan error
				
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT token, user_id, expires_at`)).
					WithArgs("user-scan").
					WillReturnRows(rows)
			},
			want:    0,
			wantErr: true,
		},
		{
			name:   "rows error after iteration",
			userID: "user-rows-err",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"token", "user_id", "expires_at", "revoked", "revoked_at",
					"user_agent", "ip_address", "created_at", "last_used_at",
				}).
					AddRow("token-1", "user-rows-err", fixedTime.Add(24*time.Hour), false, nil, nil, nil, fixedTime, fixedTime).
					RowError(0, errors.New("row error"))
				
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT token, user_id, expires_at`)).
					WithArgs("user-rows-err").
					WillReturnRows(rows)
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock database: %v", err)
			}
			defer db.Close()

			tt.setupMock(mock)
			
			repo := &RefreshTokenRepository{db: db}
			got, err := repo.GetByUserID(context.Background(), tt.userID)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByUserID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && len(got) != tt.want {
				t.Errorf("GetByUserID() returned %d tokens, want %d", len(got), tt.want)
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRefreshTokenRepository_Update(t *testing.T) {
	fixedTime := time.Now()
	
	tests := []struct {
		name      string
		token     *domain.RefreshToken
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		errType   error
	}{
		{
			name: "successful update",
			token: &domain.RefreshToken{
				Token:      "token-123",
				ExpiresAt:  fixedTime.Add(24 * time.Hour),
				Revoked:    false,
				LastUsedAt: fixedTime,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_tokens SET`)).
					WithArgs(
						"token-123",
						fixedTime.Add(24 * time.Hour),
						false,
						nil,
						fixedTime,
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "update revoked token",
			token: &domain.RefreshToken{
				Token:      "token-123",
				ExpiresAt:  fixedTime.Add(24 * time.Hour),
				Revoked:    true,
				RevokedAt:  &fixedTime,
				LastUsedAt: fixedTime,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_tokens SET`)).
					WithArgs(
						"token-123",
						fixedTime.Add(24 * time.Hour),
						true,
						fixedTime,
						fixedTime,
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "token not found",
			token: &domain.RefreshToken{
				Token:      "non-existent",
				ExpiresAt:  fixedTime.Add(24 * time.Hour),
				Revoked:    false,
				LastUsedAt: fixedTime,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_tokens SET`)).
					WithArgs(
						"non-existent",
						fixedTime.Add(24 * time.Hour),
						false,
						nil,
						fixedTime,
					).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
			errType: domain.ErrInvalidToken,
		},
		{
			name: "rows affected error",
			token: &domain.RefreshToken{
				Token:      "token-rows",
				ExpiresAt:  fixedTime.Add(24 * time.Hour),
				Revoked:    false,
				LastUsedAt: fixedTime,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_tokens SET`)).
					WithArgs(
						"token-rows",
						fixedTime.Add(24 * time.Hour),
						false,
						nil,
						fixedTime,
					).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			wantErr: true,
		},
		{
			name: "database error",
			token: &domain.RefreshToken{
				Token:      "token-123",
				ExpiresAt:  fixedTime.Add(24 * time.Hour),
				Revoked:    false,
				LastUsedAt: fixedTime,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_tokens SET`)).
					WithArgs(
						"token-123",
						fixedTime.Add(24 * time.Hour),
						false,
						nil,
						fixedTime,
					).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock database: %v", err)
			}
			defer db.Close()

			tt.setupMock(mock)
			
			repo := &RefreshTokenRepository{db: db}
			err = repo.Update(context.Background(), tt.token)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("Update() error = %v, want %v", err, tt.errType)
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRefreshTokenRepository_Revoke(t *testing.T) {
	tests := []struct {
		name       string
		tokenValue string
		setupMock  func(sqlmock.Sqlmock)
		wantErr    bool
		errType    error
	}{
		{
			name:       "successful revocation",
			tokenValue: "token-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_tokens SET`)).
					WithArgs("token-123", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:       "token not found or already revoked",
			tokenValue: "non-existent",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_tokens SET`)).
					WithArgs("non-existent", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
			errType: domain.ErrInvalidToken,
		},
		{
			name:       "rows affected error",
			tokenValue: "token-rows",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_tokens SET`)).
					WithArgs("token-rows", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			wantErr: true,
		},
		{
			name:       "database error",
			tokenValue: "token-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_tokens SET`)).
					WithArgs("token-123", sqlmock.AnyArg()).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock database: %v", err)
			}
			defer db.Close()

			tt.setupMock(mock)
			
			repo := &RefreshTokenRepository{db: db}
			err = repo.Revoke(context.Background(), tt.tokenValue)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Revoke() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("Revoke() error = %v, want %v", err, tt.errType)
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRefreshTokenRepository_RevokeAllForUser(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name:   "successful revocation",
			userID: "user-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_tokens SET`)).
					WithArgs("user-123", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 3))
			},
			wantErr: false,
		},
		{
			name:   "no tokens to revoke",
			userID: "user-456",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_tokens SET`)).
					WithArgs("user-456", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false, // This is not an error, just no tokens to revoke
		},
		{
			name:   "database error",
			userID: "user-789",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE refresh_tokens SET`)).
					WithArgs("user-789", sqlmock.AnyArg()).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock database: %v", err)
			}
			defer db.Close()

			tt.setupMock(mock)
			
			repo := &RefreshTokenRepository{db: db}
			err = repo.RevokeAllForUser(context.Background(), tt.userID)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("RevokeAllForUser() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRefreshTokenRepository_DeleteExpired(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "successful deletion",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM refresh_tokens`)).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 10))
			},
			wantErr: false,
		},
		{
			name: "no expired tokens",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM refresh_tokens`)).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name: "database error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM refresh_tokens`)).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock database: %v", err)
			}
			defer db.Close()

			tt.setupMock(mock)
			
			repo := &RefreshTokenRepository{db: db}
			err = repo.DeleteExpired(context.Background())
			
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteExpired() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRefreshTokenRepository_DeleteByToken(t *testing.T) {
	tests := []struct {
		name       string
		tokenValue string
		setupMock  func(sqlmock.Sqlmock)
		wantErr    bool
		errType    error
	}{
		{
			name:       "successful deletion",
			tokenValue: "token-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM refresh_tokens WHERE token = $1`)).
					WithArgs("token-123").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:       "token not found",
			tokenValue: "non-existent",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM refresh_tokens WHERE token = $1`)).
					WithArgs("non-existent").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
			errType: domain.ErrInvalidToken,
		},
		{
			name:       "rows affected error",
			tokenValue: "token-rows",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM refresh_tokens WHERE token = $1`)).
					WithArgs("token-rows").
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			wantErr: true,
		},
		{
			name:       "database error",
			tokenValue: "token-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM refresh_tokens WHERE token = $1`)).
					WithArgs("token-123").
					WillReturnError(errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("error creating mock database: %v", err)
			}
			defer db.Close()

			tt.setupMock(mock)
			
			repo := &RefreshTokenRepository{db: db}
			err = repo.DeleteByToken(context.Background(), tt.tokenValue)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteByToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("DeleteByToken() error = %v, want %v", err, tt.errType)
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}


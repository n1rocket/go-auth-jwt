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
	"github.com/jackc/pgx/v5/pgconn"
)

func TestNewUserRepository(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creating mock database: %v", err)
	}
	defer db.Close()

	repo := NewUserRepository(db)
	
	if repo == nil {
		t.Error("Expected repository to be created")
	}
	
	if repo.db != db {
		t.Error("Expected db to be set correctly")
	}
}

func TestUserRepository_Create(t *testing.T) {
	fixedTime := time.Now()
	
	tests := []struct {
		name    string
		user    *domain.User
		setupMock func(sqlmock.Sqlmock)
		wantErr bool
		errType error
	}{
		{
			name: "successful creation",
			user: &domain.User{
				Email:         "test@example.com",
				PasswordHash:  "hashed_password",
				EmailVerified: false,
				CreatedAt:     fixedTime,
				UpdatedAt:     fixedTime,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).
					AddRow("generated-uuid")
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users`)).
					WithArgs(
						"test@example.com",
						"hashed_password",
						false,
						nil,
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
			name: "duplicate email error",
			user: &domain.User{
				Email:         "existing@example.com",
				PasswordHash:  "hashed_password",
				EmailVerified: false,
				CreatedAt:     fixedTime,
				UpdatedAt:     fixedTime,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users`)).
					WithArgs(
						"existing@example.com",
						"hashed_password",
						false,
						nil,
						nil,
						nil,
						nil,
						fixedTime,
						fixedTime,
					).
					WillReturnError(&pgconn.PgError{
						Code: uniqueViolationCode,
					})
			},
			wantErr: true,
			errType: domain.ErrDuplicateEmail,
		},
		{
			name: "database error",
			user: &domain.User{
				Email:         "test@example.com",
				PasswordHash:  "hashed_password",
				EmailVerified: false,
				CreatedAt:     fixedTime,
				UpdatedAt:     fixedTime,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users`)).
					WithArgs(
						"test@example.com",
						"hashed_password",
						false,
						nil,
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
		{
			name: "with email verification token",
			user: &domain.User{
				Email:                      "test@example.com",
				PasswordHash:              "hashed_password",
				EmailVerified:             false,
				EmailVerificationToken:    stringPtr("verification-token"),
				EmailVerificationExpiresAt: timePtr(fixedTime.Add(24 * time.Hour)),
				CreatedAt:                 fixedTime,
				UpdatedAt:                 fixedTime,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).
					AddRow("generated-uuid")
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users`)).
					WithArgs(
						"test@example.com",
						"hashed_password",
						false,
						"verification-token",
						fixedTime.Add(24 * time.Hour),
						nil,
						nil,
						fixedTime,
						fixedTime,
					).
					WillReturnRows(rows)
			},
			wantErr: false,
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
			
			repo := &UserRepository{db: db}
			err = repo.Create(context.Background(), tt.user)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("Create() error = %v, want %v", err, tt.errType)
			}
			
			if !tt.wantErr && tt.user.ID == "" {
				t.Error("Expected user ID to be set")
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	fixedTime := time.Now()
	
	tests := []struct {
		name    string
		userID  string
		setupMock func(sqlmock.Sqlmock)
		want    *domain.User
		wantErr bool
		errType error
	}{
		{
			name:   "successful retrieval",
			userID: "user-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "email", "password_hash", "email_verified",
					"email_verification_token", "email_verification_expires_at",
					"password_reset_token", "password_reset_expires_at",
					"created_at", "updated_at",
				}).AddRow(
					"user-123", "test@example.com", "hashed_password", true,
					nil, nil, nil, nil,
					fixedTime, fixedTime,
				)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash`)).
					WithArgs("user-123").
					WillReturnRows(rows)
			},
			want: &domain.User{
				ID:            "user-123",
				Email:         "test@example.com",
				PasswordHash:  "hashed_password",
				EmailVerified: true,
				CreatedAt:     fixedTime,
				UpdatedAt:     fixedTime,
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			userID: "non-existent",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash`)).
					WithArgs("non-existent").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errType: domain.ErrUserNotFound,
		},
		{
			name:   "database error",
			userID: "user-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash`)).
					WithArgs("user-123").
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
			
			repo := &UserRepository{db: db}
			got, err := repo.GetByID(context.Background(), tt.userID)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("GetByID() error = %v, want %v", err, tt.errType)
			}
			
			if !tt.wantErr && got != nil {
				if got.ID != tt.want.ID || got.Email != tt.want.Email {
					t.Errorf("GetByID() = %v, want %v", got, tt.want)
				}
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	fixedTime := time.Now()
	
	tests := []struct {
		name    string
		email   string
		setupMock func(sqlmock.Sqlmock)
		want    *domain.User
		wantErr bool
		errType error
	}{
		{
			name:  "successful retrieval",
			email: "test@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "email", "password_hash", "email_verified",
					"email_verification_token", "email_verification_expires_at",
					"password_reset_token", "password_reset_expires_at",
					"created_at", "updated_at",
				}).AddRow(
					"user-123", "test@example.com", "hashed_password", true,
					nil, nil, nil, nil,
					fixedTime, fixedTime,
				)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash`)).
					WithArgs("test@example.com").
					WillReturnRows(rows)
			},
			want: &domain.User{
				ID:            "user-123",
				Email:         "test@example.com",
				PasswordHash:  "hashed_password",
				EmailVerified: true,
				CreatedAt:     fixedTime,
				UpdatedAt:     fixedTime,
			},
			wantErr: false,
		},
		{
			name:  "user not found",
			email: "nonexistent@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash`)).
					WithArgs("nonexistent@example.com").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errType: domain.ErrUserNotFound,
		},
		{
			name:  "database error",
			email: "test@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, email, password_hash`)).
					WithArgs("test@example.com").
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
			
			repo := &UserRepository{db: db}
			got, err := repo.GetByEmail(context.Background(), tt.email)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("GetByEmail() error = %v, want %v", err, tt.errType)
			}
			
			if !tt.wantErr && got != nil {
				if got.ID != tt.want.ID || got.Email != tt.want.Email {
					t.Errorf("GetByEmail() = %v, want %v", got, tt.want)
				}
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestUserRepository_Update(t *testing.T) {
	
	tests := []struct {
		name    string
		user    *domain.User
		setupMock func(sqlmock.Sqlmock)
		wantErr bool
		errType error
	}{
		{
			name: "successful update",
			user: &domain.User{
				ID:            "user-123",
				Email:         "updated@example.com",
				PasswordHash:  "new_hash",
				EmailVerified: true,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET`)).
					WithArgs(
						"user-123",
						"updated@example.com",
						"new_hash",
						true,
						nil,
						nil,
						nil,
						nil,
						sqlmock.AnyArg(), // updated_at
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "user not found",
			user: &domain.User{
				ID:            "non-existent",
				Email:         "test@example.com",
				PasswordHash:  "hash",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET`)).
					WithArgs(
						"non-existent",
						"test@example.com",
						"hash",
						false,
						nil,
						nil,
						nil,
						nil,
						sqlmock.AnyArg(),
					).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
			errType: domain.ErrUserNotFound,
		},
		{
			name: "duplicate email",
			user: &domain.User{
				ID:            "user-123",
				Email:         "existing@example.com",
				PasswordHash:  "hash",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET`)).
					WithArgs(
						"user-123",
						"existing@example.com",
						"hash",
						false,
						nil,
						nil,
						nil,
						nil,
						sqlmock.AnyArg(),
					).
					WillReturnError(&pgconn.PgError{
						Code: uniqueViolationCode,
					})
			},
			wantErr: true,
			errType: domain.ErrDuplicateEmail,
		},
		{
			name: "rows affected error",
			user: &domain.User{
				ID:            "user-rows",
				Email:         "test@example.com",
				PasswordHash:  "hash",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET`)).
					WithArgs(
						"user-rows",
						"test@example.com",
						"hash",
						false,
						nil,
						nil,
						nil,
						nil,
						sqlmock.AnyArg(),
					).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			wantErr: true,
		},
		{
			name: "database error",
			user: &domain.User{
				ID:            "user-123",
				Email:         "test@example.com",
				PasswordHash:  "hash",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET`)).
					WithArgs(
						"user-123",
						"test@example.com",
						"hash",
						false,
						nil,
						nil,
						nil,
						nil,
						sqlmock.AnyArg(),
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
			
			repo := &UserRepository{db: db}
			err = repo.Update(context.Background(), tt.user)
			
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

func TestUserRepository_Delete(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		setupMock func(sqlmock.Sqlmock)
		wantErr bool
		errType error
	}{
		{
			name:   "successful deletion",
			userID: "user-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM users WHERE id = $1`)).
					WithArgs("user-123").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			userID: "non-existent",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM users WHERE id = $1`)).
					WithArgs("non-existent").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
			errType: domain.ErrUserNotFound,
		},
		{
			name:   "rows affected error",
			userID: "user-rows",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM users WHERE id = $1`)).
					WithArgs("user-rows").
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			wantErr: true,
		},
		{
			name:   "database error",
			userID: "user-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM users WHERE id = $1`)).
					WithArgs("user-123").
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
			
			repo := &UserRepository{db: db}
			err = repo.Delete(context.Background(), tt.userID)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.errType != nil && !errors.Is(err, tt.errType) {
				t.Errorf("Delete() error = %v, want %v", err, tt.errType)
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		setupMock func(sqlmock.Sqlmock)
		wantExist bool
		wantErr   bool
	}{
		{
			name:  "email exists",
			email: "existing@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`)).
					WithArgs("existing@example.com").
					WillReturnRows(rows)
			},
			wantExist: true,
			wantErr:   false,
		},
		{
			name:  "email does not exist",
			email: "nonexistent@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`)).
					WithArgs("nonexistent@example.com").
					WillReturnRows(rows)
			},
			wantExist: false,
			wantErr:   false,
		},
		{
			name:  "database error",
			email: "test@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`)).
					WithArgs("test@example.com").
					WillReturnError(errors.New("database error"))
			},
			wantExist: false,
			wantErr:   true,
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
			
			repo := &UserRepository{db: db}
			exists, err := repo.ExistsByEmail(context.Background(), tt.email)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ExistsByEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if exists != tt.wantExist {
				t.Errorf("ExistsByEmail() = %v, want %v", exists, tt.wantExist)
			}
			
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}


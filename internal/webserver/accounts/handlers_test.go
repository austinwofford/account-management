package accounts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/austinwofford/account-management/internal/database"
	"github.com/austinwofford/account-management/internal/service/auth"
	"github.com/austinwofford/account-management/internal/webserver/httputils"
	"github.com/stretchr/testify/assert"
)

// Mock implementations
type mockDBRepository struct {
	createAccountFn      func(ctx context.Context, params database.AccountCreationParams) (*database.Account, error)
	getAccountFn         func(ctx context.Context, email string) (*database.Account, error)
	createRefreshTokenFn func(ctx context.Context, params database.CreateRefreshTokenParams) error
	getRefreshTokenFn    func(ctx context.Context, token string) (*database.RefreshToken, error)
	deleteRefreshTokenFn func(ctx context.Context, accountID string) error
}

func (m *mockDBRepository) CreateAccount(ctx context.Context, params database.AccountCreationParams) (*database.Account, error) {
	if m.createAccountFn != nil {
		return m.createAccountFn(ctx, params)
	}
	return &database.Account{ID: "test-id", Email: params.Email}, nil
}

func (m *mockDBRepository) GetAccount(ctx context.Context, email string) (*database.Account, error) {
	if m.getAccountFn != nil {
		return m.getAccountFn(ctx, email)
	}
	return &database.Account{ID: "test-id", Email: email, PasswordHash: "hashed-password"}, nil
}

func (m *mockDBRepository) CreateRefreshToken(ctx context.Context, params database.CreateRefreshTokenParams) error {
	if m.createRefreshTokenFn != nil {
		return m.createRefreshTokenFn(ctx, params)
	}
	return nil
}

func (m *mockDBRepository) GetRefreshToken(ctx context.Context, token string) (*database.RefreshToken, error) {
	if m.getRefreshTokenFn != nil {
		return m.getRefreshTokenFn(ctx, token)
	}
	return &database.RefreshToken{
		Token:     token,
		AccountID: "test-account-id",
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil
}

func (m *mockDBRepository) DeleteRefreshToken(ctx context.Context, accountID string) error {
	if m.deleteRefreshTokenFn != nil {
		return m.deleteRefreshTokenFn(ctx, accountID)
	}
	return nil
}

func createTestHandler(repo Repository) *handler {
	if repo == nil {
		repo = &mockDBRepository{}
	}

	return &handler{
		db:         repo,
		authClient: &auth.Client{},
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		setupMocks       func(*mockDBRepository)
		expectedStatus   int
		expectedResponse func(t *testing.T, body []byte)
	}{
		{
			name:           "valid registration",
			body:           `{"email":"test@example.com","password":"Test123!@#"}`,
			expectedStatus: http.StatusCreated,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp registerResponse
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "Account created successfully", resp.Message)
				assert.NotEmpty(t, resp.AccountID)
			},
		},
		{
			name:           "invalid JSON",
			body:           `{"email":"test@example.com","password":}`,
			expectedStatus: http.StatusBadRequest,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp httputils.ErrorResponse
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "error reading request body", resp.Message)
			},
		},
		{
			name:           "invalid email",
			body:           `{"email":"invalid-email","password":"Test123!@#"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp httputils.ErrorResponse
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, errTypeValidationError, resp.Type)
			},
		},
		{
			name:           "weak password",
			body:           `{"email":"test@example.com","password":"weak"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp httputils.ErrorResponse
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, errTypeValidationError, resp.Type)
			},
		},
		{
			name: "account already exists",
			body: `{"email":"test@example.com","password":"Test123!@#"}`,
			setupMocks: func(repo *mockDBRepository) {
				repo.createAccountFn = func(ctx context.Context, params database.AccountCreationParams) (*database.Account, error) {
					return nil, database.ErrAccountAlreadyExists
				}
			},
			expectedStatus: http.StatusConflict,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp httputils.ErrorResponse
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, errTypeAccountAlreadyExists, resp.Type)
			},
		},
		{
			name: "database error",
			body: `{"email":"test@example.com","password":"Test123!@#"}`,
			setupMocks: func(repo *mockDBRepository) {
				repo.createAccountFn = func(ctx context.Context, params database.AccountCreationParams) (*database.Account, error) {
					return nil, errors.New("database connection failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp httputils.ErrorResponse
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.Contains(t, resp.Message, "unexpected error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockDBRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(repo)
			}

			h := createTestHandler(repo)

			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.register(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedResponse != nil {
				tt.expectedResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		setupMocks       func(*mockDBRepository)
		expectedStatus   int
		expectedResponse func(t *testing.T, body []byte)
	}{
		{
			name: "valid login",
			body: `{"email":"test@example.com","password":"Test123!@#"}`,
			setupMocks: func(repo *mockDBRepository) {
				hashedPassword, err := auth.HashPassword("Test123!@#")
				assert.NoError(t, err)

				repo.getAccountFn = func(ctx context.Context, email string) (*database.Account, error) {
					return &database.Account{
						ID:           "test-account-id",
						Email:        email,
						PasswordHash: hashedPassword,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp loginOrRefreshResponse
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
				assert.Equal(t, tokenTypeBearer, resp.TokenType)
			},
		},
		{
			name:           "invalid JSON",
			body:           `{"email":"test@example.com","password":}`,
			expectedStatus: http.StatusBadRequest,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp httputils.ErrorResponse
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "error reading request body", resp.Message)
			},
		},
		{
			name: "account not found",
			body: `{"email":"nonexistent@example.com","password":"Test123!@#"}`,
			setupMocks: func(repo *mockDBRepository) {
				repo.getAccountFn = func(ctx context.Context, email string) (*database.Account, error) {
					return nil, database.ErrAccountNotFound
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp httputils.ErrorResponse
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, errTypeAccountNotFound, resp.Type)
			},
		},
		{
			name:           "wrong password",
			body:           `{"email": "test@example.com", "password": "wrongpassword"}`,
			expectedStatus: http.StatusUnauthorized,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp httputils.ErrorResponse
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, errTypeIncorrectPassword, resp.Type)
			},
		},
		{
			name: "database error",
			body: `{"email":"test@example.com","password":"Test123!@#"}`,
			setupMocks: func(repo *mockDBRepository) {
				repo.getAccountFn = func(ctx context.Context, email string) (*database.Account, error) {
					return nil, errors.New("database connection failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockDBRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(repo)
			}

			h := createTestHandler(repo)

			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.login(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedResponse != nil {
				tt.expectedResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		setupMocks       func(*mockDBRepository)
		expectedStatus   int
		expectedResponse func(t *testing.T, body []byte)
	}{
		{
			name:           "valid refresh",
			body:           `{"refresh_token":"valid-refresh-token"}`,
			expectedStatus: http.StatusOK,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp loginOrRefreshResponse
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
			},
		},
		{
			name:           "invalid JSON",
			body:           `{"refresh_token":}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "token not found",
			body: `{"refresh_token":"nonexistent-token"}`,
			setupMocks: func(repo *mockDBRepository) {
				repo.getRefreshTokenFn = func(ctx context.Context, token string) (*database.RefreshToken, error) {
					return nil, database.ErrRefreshTokenNotFound
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "expired token",
			body: `{"refresh_token":"expired-token"}`,
			setupMocks: func(repo *mockDBRepository) {
				repo.getRefreshTokenFn = func(ctx context.Context, token string) (*database.RefreshToken, error) {
					return &database.RefreshToken{
						Token:     token,
						AccountID: "test-account",
						ExpiresAt: time.Now().Add(-time.Hour), // Expired 1 hour ago
					}, nil
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockDBRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(repo)
			}

			h := createTestHandler(repo)

			req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.refresh(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedResponse != nil {
				tt.expectedResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestLogout(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		setupMocks       func(*mockDBRepository)
		expectedStatus   int
		expectedResponse func(t *testing.T, body []byte)
	}{
		{
			name:           "valid logout",
			body:           `{"refresh_token":"valid-token"}`,
			expectedStatus: http.StatusOK,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp map[string]string
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "Logged out successfully", resp["message"])
			},
		},
		{
			name:           "invalid JSON",
			body:           `{"refresh_token":}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "token not found (still succeeds)",
			body: `{"refresh_token":"nonexistent-token"}`,
			setupMocks: func(repo *mockDBRepository) {
				repo.getRefreshTokenFn = func(ctx context.Context, token string) (*database.RefreshToken, error) {
					return nil, database.ErrRefreshTokenNotFound
				}
			},
			expectedStatus: http.StatusOK,
			expectedResponse: func(t *testing.T, body []byte) {
				var resp map[string]string
				assert.NoError(t, json.Unmarshal(body, &resp))
				assert.Equal(t, "Logged out successfully", resp["message"])
			},
		},
		{
			name: "delete error",
			body: `{"refresh_token":"valid-token"}`,
			setupMocks: func(repo *mockDBRepository) {
				repo.deleteRefreshTokenFn = func(ctx context.Context, accountID string) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockDBRepository{}

			if tt.setupMocks != nil {
				tt.setupMocks(repo)
			}

			h := createTestHandler(repo)

			req := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.logout(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedResponse != nil {
				tt.expectedResponse(t, w.Body.Bytes())
			}
		})
	}
}

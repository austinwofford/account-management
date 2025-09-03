package database

import (
	"context"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()

	// Use containerized database URL (this would probably be
	// different in a CI pipeline and would need to be configurable)
	dbURL := "postgres://postgres:password@localhost:5432/account_management?sslmode=disable"

	db, err := sqlx.Connect("pgx", dbURL)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = db.Exec("DELETE FROM accounts WHERE email LIKE '%@test.com'")
	if err != nil {
		t.Fatal("Failed to clean test data:", err)
	}

	return &DB{client: db}
}

func TestCreateAccount(t *testing.T) {
	db := setupTestDB(t)

	ctx := context.Background()

	tests := []struct {
		name            string
		params          AccountCreationParams
		expectedAccount func(t *testing.T, actual Account)
		shouldError     bool
		expectedError   error
	}{
		{
			name: "successful account creation",
			params: AccountCreationParams{
				Email:        "testerbob@test.com",
				PasswordHash: "hashed-password",
			},
			expectedAccount: func(t *testing.T, actual Account) {
				assert.Equal(t, "testerbob@test.com", actual.Email)
				assert.Equal(t, "hashed-password", actual.PasswordHash)
				assert.NotZero(t, actual.CreatedAt)
				assert.NotZero(t, actual.UpdatedAt)
			},
		},
		{
			name: "duplicate email error",
			params: AccountCreationParams{
				Email:        "testerbob@test.com",
				PasswordHash: "hashed-password",
			},
			shouldError:   true,
			expectedError: ErrAccountAlreadyExists,
		},
		//{
		//	name: "empty email fails",
		//},
		//{
		//	name: "empty password hash fails",
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := db.CreateAccount(ctx, tt.params)
			if tt.shouldError {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrAccountAlreadyExists)
			} else {
				require.NoError(t, err)
				require.NotNil(t, actual)
				tt.expectedAccount(t, *actual)
			}
		})
	}

	t.Cleanup(func() {
		_, err := db.client.Exec("DELETE FROM accounts WHERE email = 'testerbob@test.com'")
		require.NoError(t, err)
		require.NoError(t, db.Close())
	})
}

func TestGetAccount(t *testing.T) {
	db := setupTestDB(t)

	ctx := context.Background()

	// Create a test account first
	testAccount, err := db.CreateAccount(ctx, AccountCreationParams{
		Email:        "gettest@test.com",
		PasswordHash: "test-password-hash",
	})
	require.NoError(t, err)

	tests := []struct {
		name            string
		email           string
		expectedAccount func(t *testing.T, actual *Account)
		shouldError     bool
		expectedError   error
	}{
		{
			name:  "successful account retrieval",
			email: "gettest@test.com",
			expectedAccount: func(t *testing.T, actual *Account) {
				assert.Equal(t, testAccount.ID, actual.ID)
				assert.Equal(t, "gettest@test.com", actual.Email)
				assert.Equal(t, "test-password-hash", actual.PasswordHash)
				assert.Equal(t, testAccount.CreatedAt, actual.CreatedAt)
				assert.Equal(t, testAccount.UpdatedAt, actual.UpdatedAt)
			},
		},
		{
			name:          "account not found error",
			email:         "nonexistent@test.com",
			shouldError:   true,
			expectedError: ErrAccountNotFound,
		},
		{
			name:          "empty email",
			email:         "",
			shouldError:   true,
			expectedError: ErrAccountNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := db.GetAccount(ctx, tt.email)
			if tt.shouldError {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
				require.Nil(t, actual)
			} else {
				require.NoError(t, err)
				require.NotNil(t, actual)
				tt.expectedAccount(t, actual)
			}
		})
	}

	t.Cleanup(func() {
		_, err := db.client.Exec("DELETE FROM accounts WHERE email = 'gettest@test.com'")
		require.NoError(t, err)
		require.NoError(t, db.Close())
	})
}

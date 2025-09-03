package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRefreshToken(t *testing.T) {
	db := setupTestDB(t)

	ctx := context.Background()

	// Create a test account first (needed for foreign key)
	testAccount, err := db.CreateAccount(ctx, AccountCreationParams{
		Email:        "tokentest@test.com",
		PasswordHash: "test-password-hash",
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		params      CreateRefreshTokenParams
		shouldError bool
	}{
		{
			name: "successful token creation",
			params: CreateRefreshTokenParams{
				Token:     "test-refresh-token-123",
				AccountID: testAccount.ID,
				ExpiresAt: time.Now().Add(time.Hour * 24),
			},
		},
		{
			name: "upsert existing token (should update)",
			params: CreateRefreshTokenParams{
				Token:     "test-refresh-token-123", // Same token
				AccountID: testAccount.ID,
				ExpiresAt: time.Now().Add(time.Hour * 48), // Different expiration
			},
		},
		{
			name: "different token for same account",
			params: CreateRefreshTokenParams{
				Token:     "test-refresh-token-456",
				AccountID: testAccount.ID,
				ExpiresAt: time.Now().Add(time.Hour * 24),
			},
		},
		{
			name: "invalid account id",
			params: CreateRefreshTokenParams{
				Token:     "test-refresh-token-invalid",
				AccountID: "non-existent-account-id",
				ExpiresAt: time.Now().Add(time.Hour * 24),
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.CreateRefreshToken(ctx, tt.params)
			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

	t.Cleanup(func() {
		_, err := db.client.Exec("DELETE FROM refresh_tokens WHERE account_id = $1", testAccount.ID)
		require.NoError(t, err)
		_, err = db.client.Exec("DELETE FROM accounts WHERE email = 'tokentest@test.com'")
		require.NoError(t, err)
		require.NoError(t, db.Close())
	})
}

func TestGetRefreshToken(t *testing.T) {
	db := setupTestDB(t)

	ctx := context.Background()

	// Create a test account first
	testAccount, err := db.CreateAccount(ctx, AccountCreationParams{
		Email:        "gettokentest@test.com",
		PasswordHash: "test-password-hash",
	})
	require.NoError(t, err)

	// Create a test refresh token
	testTokenParams := CreateRefreshTokenParams{
		Token:     "test-get-token-123",
		AccountID: testAccount.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24),
	}
	err = db.CreateRefreshToken(ctx, testTokenParams)
	require.NoError(t, err)

	tests := []struct {
		name          string
		token         string
		expectedToken func(t *testing.T, actual *RefreshToken)
		shouldError   bool
		expectedError error
	}{
		{
			name:  "successful token retrieval",
			token: "test-get-token-123",
			expectedToken: func(t *testing.T, actual *RefreshToken) {
				assert.Equal(t, "test-get-token-123", actual.Token)
				assert.Equal(t, testAccount.ID, actual.AccountID)
				assert.WithinDuration(t, testTokenParams.ExpiresAt, actual.ExpiresAt, time.Second)
				assert.NotZero(t, actual.CreatedAt)
			},
		},
		{
			name:          "token not found error",
			token:         "non-existent-token",
			shouldError:   true,
			expectedError: ErrRefreshTokenNotFound,
		},
		{
			name:          "empty token",
			token:         "",
			shouldError:   true,
			expectedError: ErrRefreshTokenNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := db.GetRefreshToken(ctx, tt.token)
			if tt.shouldError {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
				require.Nil(t, actual)
			} else {
				require.NoError(t, err)
				require.NotNil(t, actual)
				tt.expectedToken(t, actual)
			}
		})
	}

	t.Cleanup(func() {
		_, err := db.client.Exec("DELETE FROM refresh_tokens WHERE account_id = $1", testAccount.ID)
		require.NoError(t, err)
		_, err = db.client.Exec("DELETE FROM accounts WHERE email = 'gettokentest@test.com'")
		require.NoError(t, err)
		require.NoError(t, db.Close())
	})
}

func TestDeleteRefreshToken(t *testing.T) {
	db := setupTestDB(t)

	ctx := context.Background()

	// Create a test account first
	testAccount, err := db.CreateAccount(ctx, AccountCreationParams{
		Email:        "deletetokentest@test.com",
		PasswordHash: "test-password-hash",
	})
	require.NoError(t, err)

	// Create test refresh tokens
	token1Params := CreateRefreshTokenParams{
		Token:     "test-delete-token-1",
		AccountID: testAccount.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24),
	}
	token2Params := CreateRefreshTokenParams{
		Token:     "test-delete-token-2",
		AccountID: testAccount.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24),
	}

	err = db.CreateRefreshToken(ctx, token1Params)
	require.NoError(t, err)
	err = db.CreateRefreshToken(ctx, token2Params)
	require.NoError(t, err)

	tests := []struct {
		name         string
		accountID    string
		shouldError  bool
		verifyDelete func(t *testing.T)
	}{
		{
			name:      "successful token deletion",
			accountID: testAccount.ID,
			verifyDelete: func(t *testing.T) {
				// Verify tokens are deleted
				_, err := db.GetRefreshToken(ctx, "test-delete-token-1")
				require.ErrorIs(t, err, ErrRefreshTokenNotFound)
				_, err = db.GetRefreshToken(ctx, "test-delete-token-2")
				require.ErrorIs(t, err, ErrRefreshTokenNotFound)
			},
		},
		{
			name:      "delete non-existent account (should not error)",
			accountID: uuid.NewString(),
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Recreate tokens for second test
			if i == 1 {
				err = db.CreateRefreshToken(ctx, token1Params)
				require.NoError(t, err)
				err = db.CreateRefreshToken(ctx, token2Params)
				require.NoError(t, err)
			}

			err := db.DeleteRefreshToken(ctx, tt.accountID)
			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.verifyDelete != nil {
					tt.verifyDelete(t)
				}
			}
		})
	}

	t.Cleanup(func() {
		_, err := db.client.Exec("DELETE FROM refresh_tokens WHERE account_id = $1", testAccount.ID)
		require.NoError(t, err)
		_, err = db.client.Exec("DELETE FROM accounts WHERE email = 'deletetokentest@test.com'")
		require.NoError(t, err)
		require.NoError(t, db.Close())
	})
}

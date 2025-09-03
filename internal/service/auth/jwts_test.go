package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: These tests were mostly written by Claude so we should probably review them more closely!
// The value of testing the JWT library seemed questionable.

func TestNewAccessToken(t *testing.T) {
	cfg := Config{
		JWTSecretKey:           "test-secret-key",
		AccessTokenTTLMinutes:  15,
		RefreshTokenTTLMinutes: 43200, // 30 days
	}
	client := NewClient(cfg)

	tests := []struct {
		name   string
		claims Claims
	}{
		{
			name: "successful token creation",
			claims: Claims{
				AccountID: "test-account-id-123",
			},
		},
		{
			name: "token with empty account id",
			claims: Claims{
				AccountID: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime := time.Now()
			tokenString, expiresAt, err := client.NewAccessToken(tt.claims)

			require.NoError(t, err)
			assert.NotEmpty(t, tokenString)
			assert.True(t, expiresAt.After(startTime))

			// Verify token expires in approximately 15 minutes
			expectedExpiry := startTime.Add(15 * time.Minute)
			assert.WithinDuration(t, expectedExpiry, expiresAt, time.Second)

			// Parse and verify the token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Verify signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					t.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(cfg.JWTSecretKey), nil
			})

			require.NoError(t, err)
			assert.True(t, token.Valid)

			// Verify claims
			claims, ok := token.Claims.(jwt.MapClaims)
			require.True(t, ok)

			assert.Equal(t, tt.claims.AccountID, claims["account_id"])
			assert.Equal(t, "account-management", claims["iss"])
			assert.NotEmpty(t, claims["jti"])

			// Verify timing claims
			iat, ok := claims["iat"].(float64)
			require.True(t, ok)
			assert.WithinDuration(t, startTime, time.Unix(int64(iat), 0), time.Second)

			exp, ok := claims["exp"].(float64)
			require.True(t, ok)
			assert.WithinDuration(t, expiresAt, time.Unix(int64(exp), 0), time.Second)
		})
	}
}

func TestNewAccessTokenWithInvalidSecret(t *testing.T) {
	cfg := Config{
		JWTSecretKey:           "test-secret-key",
		AccessTokenTTLMinutes:  15,
		RefreshTokenTTLMinutes: 43200,
	}
	client := NewClient(cfg)

	claims := Claims{
		AccountID: "test-account-id",
	}

	tokenString, _, err := client.NewAccessToken(claims)
	require.NoError(t, err)

	// Try to parse with wrong secret
	_, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("wrong-secret"), nil
	})

	assert.Error(t, err)
}

func TestNewRefreshToken(t *testing.T) {
	cfg := Config{
		JWTSecretKey:           "test-secret-key",
		AccessTokenTTLMinutes:  15,
		RefreshTokenTTLMinutes: 43200, // 30 days
	}
	client := NewClient(cfg)

	startTime := time.Now()
	token, expiresAt := client.NewRefreshToken()

	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(startTime))

	// Verify token expires in approximately 30 days
	expectedExpiry := startTime.Add(time.Duration(cfg.RefreshTokenTTLMinutes) * time.Minute)
	assert.WithinDuration(t, expectedExpiry, expiresAt, time.Second)

	// Verify token is a valid UUID format
	assert.Len(t, token, 36) // UUID length with hyphens
	assert.Contains(t, token, "-")
}

func TestNewRefreshTokenUniqueness(t *testing.T) {
	cfg := Config{
		JWTSecretKey:           "test-secret-key",
		AccessTokenTTLMinutes:  15,
		RefreshTokenTTLMinutes: 43200,
	}
	client := NewClient(cfg)

	// Generate multiple tokens and ensure they're unique
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, _ := client.NewRefreshToken()
		if tokens[token] {
			t.Errorf("duplicate refresh token generated: %s", token)
		}
		tokens[token] = true
	}
}

func TestClientWithDifferentTTL(t *testing.T) {
	tests := []struct {
		name                   string
		accessTokenTTLMinutes  int
		refreshTokenTTLMinutes int
	}{
		{
			name:                   "1 minute access, 1 hour refresh",
			accessTokenTTLMinutes:  1,
			refreshTokenTTLMinutes: 60,
		},
		{
			name:                   "60 minute access, 7 days refresh",
			accessTokenTTLMinutes:  60,
			refreshTokenTTLMinutes: 10080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				JWTSecretKey:           "test-secret-key",
				AccessTokenTTLMinutes:  tt.accessTokenTTLMinutes,
				RefreshTokenTTLMinutes: tt.refreshTokenTTLMinutes,
			}
			client := NewClient(cfg)

			startTime := time.Now()

			// Test access token TTL
			_, accessExpiresAt, err := client.NewAccessToken(Claims{
				AccountID: "test-account",
			})
			require.NoError(t, err)

			expectedAccessExpiry := startTime.Add(time.Duration(tt.accessTokenTTLMinutes) * time.Minute)
			assert.WithinDuration(t, expectedAccessExpiry, accessExpiresAt, time.Second)

			// Test refresh token TTL
			_, refreshExpiresAt := client.NewRefreshToken()
			expectedRefreshExpiry := startTime.Add(time.Duration(tt.refreshTokenTTLMinutes) * time.Minute)
			assert.WithinDuration(t, expectedRefreshExpiry, refreshExpiresAt, time.Second)
		})
	}
}

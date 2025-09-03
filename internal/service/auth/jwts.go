package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TODO: This package is a little strange having some basic email/password validation
// and the "auth" client. Should probably be moved/broken up/renamed.

type Client struct {
	jwtSecretKey           string
	accessTokenTTLMinutes  int
	refreshTokenTTLMinutes int
}

type Config struct {
	JWTSecretKey           string
	AccessTokenTTLMinutes  int
	RefreshTokenTTLMinutes int
}

func NewClient(cfg Config) *Client {
	return &Client{
		jwtSecretKey:           cfg.JWTSecretKey,
		accessTokenTTLMinutes:  cfg.AccessTokenTTLMinutes,
		refreshTokenTTLMinutes: cfg.RefreshTokenTTLMinutes,
	}
}

type Claims struct {
	AccountID string `json:"account_id"`
}

// NewAccessToken returns a signed JWT string and the expiration time (or an error)
func (c *Client) NewAccessToken(claims Claims) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(time.Minute * time.Duration(c.accessTokenTTLMinutes))

	myClaims := struct {
		Claims
		jwt.RegisteredClaims
	}{
		Claims: claims,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "account-management",
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, myClaims)

	signedToken, err := token.SignedString([]byte(c.jwtSecretKey))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("error signing token: %w", err)
	}

	return signedToken, expiresAt, nil
}

// NewRefreshToken returns a refresh token and its expiration time
func (c *Client) NewRefreshToken() (string, time.Time) {
	return uuid.NewString(), time.Now().Add(time.Duration(c.refreshTokenTTLMinutes) * time.Minute)
}

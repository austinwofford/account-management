package auth

import (
	"fmt"
	"net/mail"
	"regexp"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("Validation Error: %s", e.Message)
}

func NewValidationError(message string) ValidationError {
	return ValidationError{Message: message}
}

type Client struct {
	jwtSecretKey           string
	accessTokenTTLMinutes  int
	refreshTokenTTLMinutes int
}

type Config struct {
	JWTSecretKey            string
	AccessTokenTTLMinutes   int
	RefresnhTokenTTLMinutes int
}

func NewClient(cfg Config) *Client {
	return &Client{
		jwtSecretKey:           cfg.JWTSecretKey,
		accessTokenTTLMinutes:  cfg.AccessTokenTTLMinutes,
		refreshTokenTTLMinutes: cfg.RefresnhTokenTTLMinutes,
	}
}

type AccountManagementClaims struct {
	AccountID string `json:"account_id"`
}

// NewAccessToken returns a signed JWT string and the expiration time (or an error)
func (c *Client) NewAccessToken(claims AccountManagementClaims) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(time.Minute * time.Duration(c.accessTokenTTLMinutes))

	myClaims := struct {
		AccountManagementClaims
		jwt.StandardClaims
	}{
		AccountManagementClaims: claims,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt.Unix(),
			IssuedAt:  now.Unix(),
			Issuer:    "account-management",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, myClaims)

	signedToken, err := token.SignedString([]byte(c.jwtSecretKey))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("error signing token: %w", err)
	}

	return signedToken, expiresAt, nil
}

// NewRefreshToken returns a refrersh token and its expiration time
func (c *Client) NewRefreshToken() (string, time.Time) {
	return uuid.NewString(), time.Now().Add(time.Duration(c.refreshTokenTTLMinutes) * time.Minute)
}

func HashPassword(password string) (string, error) {
	err := validatePassword(password)
	if err != nil {
		return "", err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (c *Client) GetAccessTokenTTLSeconds() int {
	return c.accessTokenTTLMinutes * 60
}

func PasswordIsCorrect(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return NewValidationError("password must be at least 8 characters long")
	}

	// 72 characters is the max length that bcrypt will handle
	if len(password) > 72 {
		return NewValidationError("password must be less than or equal to 72 characters long")
	}

	// Check for at least one uppercase, lowercase, digit, and special character
	patterns := []string{
		`[a-z]`,      // lowercase
		`[A-Z]`,      // uppercase
		`[0-9]`,      // digit
		`[!@#$%^&*]`, // special characters
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, password)
		if !matched {
			return NewValidationError("password must contain uppercase, lowercase, digit, and special character")
		}
	}

	return nil
}

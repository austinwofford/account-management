package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid password",
			password: "Password123!",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "Pass1!",
			wantErr:  true,
			errMsg:   "password must be at least 8 characters long",
		},
		{
			name:     "too long",
			password: "Password123!" + "a123456789012345678901234567890123456789012345678901234567890",
			wantErr:  true,
			errMsg:   "password must be less than or equal to 72 characters long",
		},
		{
			name:     "no uppercase",
			password: "password123!",
			wantErr:  true,
			errMsg:   "password must contain uppercase, lowercase, digit, and special character",
		},
		{
			name:     "no lowercase",
			password: "PASSWORD123!",
			wantErr:  true,
			errMsg:   "password must contain uppercase, lowercase, digit, and special character",
		},
		{
			name:     "no digit",
			password: "Password!",
			wantErr:  true,
			errMsg:   "password must contain uppercase, lowercase, digit, and special character",
		},
		{
			name:     "no special character",
			password: "Password123",
			wantErr:  true,
			errMsg:   "password must contain uppercase, lowercase, digit, and special character",
		},
		{
			name:     "all requirements met with different special chars",
			password: "Password123@",
			wantErr:  false,
		},
		{
			name:     "minimum valid length",
			password: "Aa1!bcde",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := validatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, actual)
				expected := "Validation Error: " + tt.errMsg
				assert.Equal(t, expected, actual.Error())
			} else {
				assert.NoError(t, actual)
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "Password123!",
			wantErr:  false,
		},
		{
			name:     "invalid password - too short",
			password: "Pass1!",
			wantErr:  true,
		},
		{
			name:     "invalid password - no special char",
			password: "Password123",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := HashPassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, actual)
				assert.NotEqual(t, tt.password, actual)
				// Verify the hash is valid by trying to compare it
				assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(actual), []byte(tt.password)))
			}
		})
	}
}

func TestPasswordIsCorrect(t *testing.T) {
	validPassword := "Password123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(validPassword), bcrypt.DefaultCost)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		password       string
		hashedPassword string
		expected       bool
	}{
		{
			name:           "correct password",
			password:       validPassword,
			hashedPassword: string(hashedPassword),
			expected:       true,
		},
		{
			name:           "incorrect password",
			password:       "WrongPassword123!",
			hashedPassword: string(hashedPassword),
			expected:       false,
		},
		{
			name:           "empty password",
			password:       "",
			hashedPassword: string(hashedPassword),
			expected:       false,
		},
		{
			name:           "invalid hash",
			password:       validPassword,
			hashedPassword: "invalid-hash",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := PasswordIsCorrect(tt.password, tt.hashedPassword)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{
			name:     "valid email",
			email:    "user@example.com",
			expected: true,
		},
		{
			name:     "valid email with subdomain",
			email:    "user@mail.example.com",
			expected: true,
		},
		{
			name:     "valid email with plus",
			email:    "user+tag@example.com",
			expected: true,
		},
		{
			name:     "valid email with dots",
			email:    "user.name@example.com",
			expected: true,
		},
		{
			name:     "invalid email - no @",
			email:    "userexample.com",
			expected: false,
		},
		{
			name:     "invalid email - no domain",
			email:    "user@",
			expected: false,
		},
		{
			name:     "invalid email - no user",
			email:    "@example.com",
			expected: false,
		},
		{
			name:     "invalid email - multiple @",
			email:    "user@@example.com",
			expected: false,
		},
		{
			name:     "empty email",
			email:    "",
			expected: false,
		},
		{
			name:     "just @",
			email:    "@",
			expected: false,
		},
		{
			name:     "spaces in email",
			email:    "user name@example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := IsValidEmail(tt.email)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

package auth

import (
	"fmt"
	"net/mail"
	"regexp"

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

func PasswordIsCorrect(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// Check for at least one uppercase, lowercase, digit, and special character
var passwordPatterns = []*regexp.Regexp{
	regexp.MustCompile(`[a-z]`),
	regexp.MustCompile(`[A-Z]`),
	regexp.MustCompile(`[0-9]`),
	regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`),
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return NewValidationError("password must be at least 8 characters long")
	}

	// 72 characters is the max length that bcrypt will handle
	if len(password) > 72 {
		return NewValidationError("password must be less than or equal to 72 characters long")
	}

	for _, pattern := range passwordPatterns {
		matched := pattern.MatchString(password)
		if !matched {
			return NewValidationError("password must contain uppercase, lowercase, digit, and special character")
		}
	}

	return nil
}

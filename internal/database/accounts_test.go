package database

import (
	"context"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()

	// Use test database URL or skip if not available
	dbURL := "postgres://postgres:password@localhost:5432/account_management_test?sslmode=disable"

	db, err := sqlx.Connect("pgx", dbURL)
	if err != nil {
		t.Skip("Test database not available:", err)
	}

	// Clean up any existing test data
	_, err = db.Exec("DELETE FROM accounts WHERE email LIKE '%@test.com'")
	if err != nil {
		t.Fatal("Failed to clean test data:", err)
	}

	return &DB{client: db}
}

func TestCreateAccount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	t.Run("successful account creation", func(t *testing.T) {
		account := AccountCreationParams{
			Email:        "test@test.com",
			PasswordHash: "hashed_password",
		}

		result, err := db.CreateAccount(ctx, account)
		if err != nil {
			t.Fatal("CreateAccount failed:", err)
		}

		if result == nil {
			t.Fatal("Expected account result, got nil")
		}

		if result.ID == "" {
			t.Error("Expected ID to be set")
		}

		if result.Email != account.Email {
			t.Errorf("Expected email %s, got %s", account.Email, result.Email)
		}

		if result.PasswordHash != account.PasswordHash {
			t.Errorf("Expected password hash %s, got %s", account.PasswordHash, result.PasswordHash)
		}

		// Clean up
		_, err = db.client.Exec("DELETE FROM accounts WHERE id = $1", result.ID)
		if err != nil {
			t.Log("Failed to clean up test account:", err)
		}
	})

	t.Run("duplicate email should fail", func(t *testing.T) {
		account := AccountCreationParams{
			Email:        "duplicate@test.com",
			PasswordHash: "hashed_password",
		}

		// Create first account
		result1, err := db.CreateAccount(ctx, account)
		if err != nil {
			t.Fatal("First CreateAccount failed:", err)
		}

		// Try to create duplicate
		_, err = db.CreateAccount(ctx, account)
		if err == nil {
			t.Error("Expected error for duplicate email, got nil")
		}

		// Clean up
		_, err = db.client.Exec("DELETE FROM accounts WHERE id = $1", result1.ID)
		if err != nil {
			t.Log("Failed to clean up test account:", err)
		}
	})

	t.Run("empty email should fail", func(t *testing.T) {
		account := AccountCreationParams{
			Email:        "",
			PasswordHash: "hashed_password",
		}

		_, err := db.CreateAccount(ctx, account)
		if err == nil {
			t.Error("Expected error for empty email, got nil")
		}
	})

	t.Run("empty password hash should fail", func(t *testing.T) {
		account := AccountCreationParams{
			Email:        "empty_password@test.com",
			PasswordHash: "",
		}

		_, err := db.CreateAccount(ctx, account)
		if err == nil {
			t.Error("Expected error for empty password hash, got nil")
		}
	})
}

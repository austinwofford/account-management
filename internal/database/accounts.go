package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type Account struct {
	ID           string    `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type AccountCreationParams struct {
	Email        string `db:"email" json:"-"`
	PasswordHash string `db:"password_hash" json:"-"`
}

var (
	ErrAccountNotFound      = errors.New("an account with this email was not found")
	ErrAccountAlreadyExists = errors.New("account with this email already exists")

	duplicateEmailConstraint = "accounts_email_key"
)

func uniqueConstraint(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return pgErr.ConstraintName, true
	}
	return "", false
}

func (d *DB) CreateAccount(ctx context.Context, params AccountCreationParams) (*Account, error) {
	rows, err := d.client.NamedQueryContext(ctx, createAccountSQL, params)
	if err != nil {
		// Check for unique constraint violation
		if c, _ := uniqueConstraint(err); c == duplicateEmailConstraint {
			return nil, ErrAccountAlreadyExists
		}
		return nil, fmt.Errorf("error executing create account query: %w", err)
	}
	defer rows.Close()

	var result Account
	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	if err := rows.StructScan(&result); err != nil {
		return nil, fmt.Errorf("error scanning created account result: %w", err)
	}

	return &result, nil
}

func (d *DB) GetAccount(ctx context.Context, email string) (*Account, error) {
	var result Account
	err := d.client.GetContext(ctx, &result, getAccountSQL, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("error getting account: %w", err)
	}

	return &result, nil
}

var (
	createAccountSQL = `
		INSERT INTO accounts (email, password_hash)
		VALUES (:email, :password_hash)
		RETURNING id, email, password_hash, created_at, updated_at;`

	getAccountSQL = `
		SELECT id, email, password_hash, created_at, updated_at
		FROM accounts WHERE email = $1;`
)

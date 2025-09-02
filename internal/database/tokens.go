package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

type CreateRefreshTokenParams struct {
	Token     string    `db:"token"`
	AccountID string    `db:"account_id"`
	ExpiresAt time.Time `db:"expires_at"`
}

func (d *DB) CreateRefreshToken(ctx context.Context, params CreateRefreshTokenParams) error {
	_, err := d.client.NamedExecContext(ctx, createRefreshTokenSQL, params)
	if err != nil {
		return fmt.Errorf("error creating refresh token: %w", err)
	}
	return nil
}

type RefreshToken struct {
	Token     string    `db:"token"`
	AccountID string    `db:"account_id"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

func (d *DB) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	var result RefreshToken
	err := d.client.GetContext(ctx, &result, getRefreshTokenSQL, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("error getting refresh token: %w", err)
	}
	return &result, nil
}

func (d *DB) DeleteRefreshToken(ctx context.Context, accountID string) error {
	_, err := d.client.ExecContext(ctx, deleteRefreshTokenSQL, accountID)
	if err != nil {
		return fmt.Errorf("error deleting refresh token: %w", err)
	}
	return nil
}

var (
	createRefreshTokenSQL = `
		INSERT INTO refresh_tokens (token, account_id, expires_at)
		VALUES (:token, :account_id, :expires_at)
		ON CONFLICT (token) 
		DO UPDATE SET 
			token = EXCLUDED.token,
			expires_at = EXCLUDED.expires_at,
			created_at = NOW();`

	getRefreshTokenSQL = `
		SELECT token, account_id, expires_at, created_at
		FROM refresh_tokens 
		WHERE token = $1;`

	deleteRefreshTokenSQL = `
		DELETE FROM refresh_tokens 
		WHERE account_id = $1;`
)

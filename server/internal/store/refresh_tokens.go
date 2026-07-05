package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"project-prism/server/internal/models"
)

type RefreshTokenSession struct {
	Token *models.RefreshToken
	User  *models.User
}

type RefreshTokenStore struct {
	pool DB
}

func NewRefreshTokenStore(pool DB) *RefreshTokenStore {
	return &RefreshTokenStore{pool: pool}
}

func (s *RefreshTokenStore) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*models.RefreshToken, error) {
	const q = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, created_at, updated_at, last_used_at, revoked_at`

	token := &models.RefreshToken{}
	err := s.pool.QueryRow(ctx, q, userID, tokenHash, expiresAt).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.UpdatedAt,
		&token.LastUsedAt,
		&token.RevokedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert refresh token: %w", err)
	}

	return token, nil
}

func (s *RefreshTokenStore) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*RefreshTokenSession, error) {
	const q = `
		SELECT
			rt.id,
			rt.user_id,
			rt.token_hash,
			rt.expires_at,
			rt.created_at,
			rt.updated_at,
			rt.last_used_at,
			rt.revoked_at,
			u.id,
			u.email,
			u.password_hash,
			u.role,
			u.status,
			u.created_at,
			u.updated_at,
			u.last_login_at
		FROM refresh_tokens rt
		INNER JOIN users u ON u.id = rt.user_id
		WHERE rt.token_hash = $1
		  AND rt.revoked_at IS NULL
		  AND rt.expires_at > NOW()`

	token := &models.RefreshToken{}
	user := &models.User{}
	err := s.pool.QueryRow(ctx, q, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.UpdatedAt,
		&token.LastUsedAt,
		&token.RevokedAt,
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get refresh token session: %w", err)
	}

	return &RefreshTokenSession{
		Token: token,
		User:  user,
	}, nil
}

func (s *RefreshTokenStore) Rotate(ctx context.Context, currentTokenHash, nextTokenHash string, expiresAt time.Time) (*models.RefreshToken, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin refresh rotation: %w", err)
	}
	defer tx.Rollback(ctx)

	const lockCurrent = `
		SELECT user_id
		FROM refresh_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		FOR UPDATE`

	var userID string
	if err := tx.QueryRow(ctx, lockCurrent, currentTokenHash).Scan(&userID); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("lock refresh token: %w", err)
	}

	const revokeCurrent = `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), last_used_at = NOW(), updated_at = NOW()
		WHERE token_hash = $1`

	if _, err := tx.Exec(ctx, revokeCurrent, currentTokenHash); err != nil {
		return nil, fmt.Errorf("revoke refresh token: %w", err)
	}

	const createNext = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, created_at, updated_at, last_used_at, revoked_at`

	token := &models.RefreshToken{}
	if err := tx.QueryRow(ctx, createNext, userID, nextTokenHash, expiresAt).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.UpdatedAt,
		&token.LastUsedAt,
		&token.RevokedAt,
	); err != nil {
		return nil, fmt.Errorf("create rotated refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit refresh rotation: %w", err)
	}

	return token, nil
}

func (s *RefreshTokenStore) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), updated_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL`

	if _, err := s.pool.Exec(ctx, q, tokenHash); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func (s *RefreshTokenStore) RevokeAllByUserID(ctx context.Context, userID string) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL`

	if _, err := s.pool.Exec(ctx, q, userID); err != nil {
		return fmt.Errorf("revoke refresh tokens by user id: %w", err)
	}

	return nil
}

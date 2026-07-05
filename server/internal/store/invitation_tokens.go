package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"project-prism/server/internal/models"
)

type InvitationTokenSession struct {
	Token *models.InvitationToken
	User  *models.User
}

type InvitationTokenStore struct {
	pool DB
}

func NewInvitationTokenStore(pool DB) *InvitationTokenStore {
	return &InvitationTokenStore{pool: pool}
}

func (s *InvitationTokenStore) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*models.InvitationToken, error) {
	const q = `
		INSERT INTO invitation_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, created_at, consumed_at`

	token := &models.InvitationToken{}
	err := s.pool.QueryRow(ctx, q, userID, tokenHash, expiresAt).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.ConsumedAt,
	)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("insert invitation token: %w", err)
	}

	return token, nil
}

func (s *InvitationTokenStore) GetByTokenHash(ctx context.Context, tokenHash string) (*InvitationTokenSession, error) {
	const q = `
		SELECT
			it.id,
			it.user_id,
			it.token_hash,
			it.expires_at,
			it.created_at,
			it.consumed_at,
			u.id,
			u.email,
			u.password_hash,
			u.role,
			u.status,
			u.created_at,
			u.updated_at,
			u.last_login_at
		FROM invitation_tokens it
		INNER JOIN users u ON u.id = it.user_id
		WHERE it.token_hash = $1
		  AND it.consumed_at IS NULL
		  AND it.expires_at > NOW()
		  AND u.status = $2`

	token := &models.InvitationToken{}
	user := &models.User{}
	err := s.pool.QueryRow(ctx, q, tokenHash, models.UserStatusInvited).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.ConsumedAt,
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
		return nil, fmt.Errorf("get invitation token: %w", err)
	}

	return &InvitationTokenSession{
		Token: token,
		User:  user,
	}, nil
}

func (s *InvitationTokenStore) ActivateByTokenHash(ctx context.Context, tokenHash, passwordHash string) (*models.User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin invitation activation: %w", err)
	}
	defer tx.Rollback(ctx)

	const lockInvitation = `
		SELECT user_id
		FROM invitation_tokens
		WHERE token_hash = $1
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
		FOR UPDATE`

	var userID string
	if err := tx.QueryRow(ctx, lockInvitation, tokenHash).Scan(&userID); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("lock invitation token: %w", err)
	}

	const activateUser = `
		UPDATE users
		SET password_hash = $1, status = $2, updated_at = NOW()
		WHERE id = $3 AND status = $4
		RETURNING id, email, password_hash, role, status, created_at, updated_at, last_login_at`

	user := &models.User{}
	if err := tx.QueryRow(ctx, activateUser, passwordHash, models.UserStatusActive, userID, models.UserStatusInvited).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("activate invited user: %w", err)
	}

	const consumeInvitation = `
		UPDATE invitation_tokens
		SET consumed_at = NOW()
		WHERE token_hash = $1`

	if _, err := tx.Exec(ctx, consumeInvitation, tokenHash); err != nil {
		return nil, fmt.Errorf("consume invitation token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit invitation activation: %w", err)
	}

	return user, nil
}

func (s *InvitationTokenStore) DeleteByUserID(ctx context.Context, userID string) error {
	const q = `
		DELETE FROM invitation_tokens
		WHERE user_id = $1`

	if _, err := s.pool.Exec(ctx, q, userID); err != nil {
		return fmt.Errorf("delete invitation tokens by user id: %w", err)
	}

	return nil
}

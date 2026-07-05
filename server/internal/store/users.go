package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"project-prism/server/internal/models"
)

type UserStore struct {
	pool DB
}

func NewUserStore(pool DB) *UserStore {
	return &UserStore{pool: pool}
}

func (s *UserStore) Create(ctx context.Context, email string, passwordHash *string, role models.UserRole, status models.UserStatus) (*models.User, error) {
	const q = `
		INSERT INTO users (email, password_hash, role, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, password_hash, role, status, created_at, updated_at, last_login_at`

	user := &models.User{}
	err := s.pool.QueryRow(ctx, q, email, passwordHash, role, status).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

func (s *UserStore) List(ctx context.Context) ([]*models.User, error) {
	const q = `
		SELECT id, email, role, status, created_at, updated_at, last_login_at
		FROM users
		ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Role,
			&user.Status,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.LastLoginAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users rows: %w", err)
	}

	return users, nil
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*models.User, error) {
	const q = `
		SELECT id, email, password_hash, role, status, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1`

	user := &models.User{}
	err := s.pool.QueryRow(ctx, q, id).Scan(
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
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return user, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	const q = `
		SELECT id, email, password_hash, role, status, created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1`

	user := &models.User{}
	err := s.pool.QueryRow(ctx, q, email).Scan(
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
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}

func (s *UserStore) UpdateStatus(ctx context.Context, id string, status models.UserStatus) (*models.User, error) {
	const q = `
		UPDATE users
		SET status = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, email, password_hash, role, status, created_at, updated_at, last_login_at`

	user := &models.User{}
	err := s.pool.QueryRow(ctx, q, status, id).Scan(
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
		return nil, fmt.Errorf("update user status: %w", err)
	}

	return user, nil
}

func (s *UserStore) TouchLastLogin(ctx context.Context, id string) error {
	const q = `
		UPDATE users
		SET last_login_at = NOW(), updated_at = NOW()
		WHERE id = $1`

	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("touch last login: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *UserStore) CountActiveAdmins(ctx context.Context) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM users
		WHERE role = $1 AND status = $2`

	var count int
	if err := s.pool.QueryRow(ctx, q, models.UserRoleAdmin, models.UserStatusActive).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active admins: %w", err)
	}

	return count, nil
}

func (s *UserStore) UpsertBootstrapAdmin(ctx context.Context, email, passwordHash string) (*models.User, error) {
	const q = `
		INSERT INTO users (email, password_hash, role, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
		    role = EXCLUDED.role,
		    status = EXCLUDED.status,
		    updated_at = NOW()
		RETURNING id, email, password_hash, role, status, created_at, updated_at, last_login_at`

	user := &models.User{}
	err := s.pool.QueryRow(ctx, q, email, passwordHash, models.UserRoleAdmin, models.UserStatusActive).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert bootstrap admin: %w", err)
	}

	return user, nil
}

func (s *UserStore) DeleteInvitedByID(ctx context.Context, id string) error {
	const q = `
		DELETE FROM users
		WHERE id = $1 AND status = $2`

	tag, err := s.pool.Exec(ctx, q, id, models.UserStatusInvited)
	if err != nil {
		return fmt.Errorf("delete invited user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

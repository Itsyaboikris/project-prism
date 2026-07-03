package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"project-prism/server/internal/models"
)

var ErrNotFound = errors.New("record not found")

type ApplicationStore struct {
	pool *pgxpool.Pool
}

func NewApplicationStore(pool *pgxpool.Pool) *ApplicationStore {
	return &ApplicationStore{pool: pool}
}

func (s *ApplicationStore) Create(ctx context.Context, name, apiKey string) (*models.Application, error) {
	const q = `
		INSERT INTO applications (name, api_key)
		VALUES ($1, $2)
		RETURNING id, name, api_key, created_at, updated_at`

	app := &models.Application{}
	err := s.pool.QueryRow(ctx, q, name, apiKey).Scan(
		&app.ID,
		&app.Name,
		&app.APIKey,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert application: %w", err)
	}

	return app, nil
}

func (s *ApplicationStore) List(ctx context.Context) ([]*models.Application, error) {
	const q = `
		SELECT id, name, api_key, created_at, updated_at
		FROM applications
		ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var apps []*models.Application
	for rows.Next() {
		app := &models.Application{}
		if err := rows.Scan(&app.ID, &app.Name, &app.APIKey, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list applications rows: %w", err)
	}

	return apps, nil
}

func (s *ApplicationStore) GetByID(ctx context.Context, id string) (*models.Application, error) {
	const q = `
		SELECT id, name, api_key, created_at, updated_at
		FROM applications
		WHERE id = $1`

	app := &models.Application{}
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&app.ID,
		&app.Name,
		&app.APIKey,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get application: %w", err)
	}

	return app, nil
}

func (s *ApplicationStore) Update(ctx context.Context, id, name string) (*models.Application, error) {
	const q = `
		UPDATE applications
		SET name = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, name, api_key, created_at, updated_at`

	app := &models.Application{}
	err := s.pool.QueryRow(ctx, q, name, id).Scan(
		&app.ID,
		&app.Name,
		&app.APIKey,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update application: %w", err)
	}

	return app, nil
}

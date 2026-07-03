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
var ErrInactive = errors.New("record inactive")

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
		RETURNING id, name, api_key, status, created_at, updated_at`

	app := &models.Application{}
	err := s.pool.QueryRow(ctx, q, name, apiKey).Scan(
		&app.ID,
		&app.Name,
		&app.APIKey,
		&app.Status,
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
		SELECT id, name, api_key, status, created_at, updated_at
		FROM applications
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var apps []*models.Application
	for rows.Next() {
		app := &models.Application{}
		if err := rows.Scan(&app.ID, &app.Name, &app.APIKey, &app.Status, &app.CreatedAt, &app.UpdatedAt); err != nil {
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
		SELECT id, name, api_key, status, created_at, updated_at
		FROM applications
		WHERE id = $1 AND deleted_at IS NULL`

	app := &models.Application{}
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&app.ID,
		&app.Name,
		&app.APIKey,
		&app.Status,
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

type UpdateApplicationParams struct {
	Name   string
	Status *models.ApplicationStatus
}

func (s *ApplicationStore) Update(ctx context.Context, id string, p UpdateApplicationParams) (*models.Application, error) {
	const q = `
		UPDATE applications
		SET name = $1, status = COALESCE($2, status), updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING id, name, api_key, status, created_at, updated_at`

	app := &models.Application{}
	err := s.pool.QueryRow(ctx, q, p.Name, p.Status, id).Scan(
		&app.ID,
		&app.Name,
		&app.APIKey,
		&app.Status,
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

func (s *ApplicationStore) Delete(ctx context.Context, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const softDeleteBranches = `
		UPDATE branches b
		SET deleted_at = NOW()
		FROM experiments e
		WHERE b.experiment_id = e.id
		  AND e.application_id = $1
		  AND b.deleted_at IS NULL
		  AND e.deleted_at IS NULL`

	if _, err := tx.Exec(ctx, softDeleteBranches, id); err != nil {
		return fmt.Errorf("delete application branches: %w", err)
	}

	const softDeleteExperiments = `
		UPDATE experiments
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE application_id = $1 AND deleted_at IS NULL`

	if _, err := tx.Exec(ctx, softDeleteExperiments, id); err != nil {
		return fmt.Errorf("delete application experiments: %w", err)
	}

	const softDeleteApplication = `
		UPDATE applications
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	tag, err := tx.Exec(ctx, softDeleteApplication, id)
	if err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

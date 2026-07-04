package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"project-prism/server/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrConflict = errors.New("record already exists")

type CreateExperimentParams struct {
	ApplicationID string
	Key           string
	Name          string
	Description   *string
	StartDate     *time.Time
	EndDate       *time.Time
	Branches      []CreateBranchParams
}

type UpdateExperimentParams struct {
	Name        string
	Description *string
	Status      models.ExperimentStatus
	StartDate   *time.Time
	EndDate     *time.Time
}

type ExperimentStore struct {
	pool DB
}

func NewExperimentStore(pool DB) *ExperimentStore {
	return &ExperimentStore{pool: pool}
}

func (s *ExperimentStore) Create(ctx context.Context, p CreateExperimentParams) (*models.Experiment, error) {
	if err := s.ensureApplicationActive(ctx, s.pool, p.ApplicationID); err != nil {
		return nil, err
	}

	if len(p.Branches) > 0 {
		return s.createWithBranches(ctx, p)
	}
	return s.createExperiment(ctx, s.pool, p)
}

func (s *ExperimentStore) ensureApplicationActive(ctx context.Context, db pgxQuerier, applicationID string) error {
	const query = `
		SELECT status
		FROM applications
		WHERE id = $1 AND deleted_at IS NULL`

	var status models.ApplicationStatus
	err := db.QueryRow(ctx, query, applicationID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("verify application: %w", err)
	}
	if status != models.ApplicationStatusActive {
		return ErrInactive
	}

	return nil
}

func (s *ExperimentStore) createExperiment(ctx context.Context, q pgxQuerier, p CreateExperimentParams) (*models.Experiment, error) {
	const query = `
		INSERT INTO experiments (application_id, key, name, description, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, application_id, key, name, description, status, start_date, end_date, created_at, updated_at`

	exp := &models.Experiment{Branches: []*models.Branch{}}
	err := q.QueryRow(ctx, query,
		p.ApplicationID, p.Key, p.Name, p.Description, p.StartDate, p.EndDate,
	).Scan(
		&exp.ID, &exp.ApplicationID, &exp.Key, &exp.Name, &exp.Description,
		&exp.Status, &exp.StartDate, &exp.EndDate, &exp.CreatedAt, &exp.UpdatedAt,
	)
	if err != nil {
		return nil, classifyExperimentErr("create experiment", err)
	}

	return exp, nil
}

func (s *ExperimentStore) createWithBranches(ctx context.Context, p CreateExperimentParams) (*models.Experiment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	exp, err := s.createExperiment(ctx, tx, p)
	if err != nil {
		return nil, err
	}

	const branchQ = `
		INSERT INTO branches (experiment_id, key, name, weight, metadata_json)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, experiment_id, key, name, weight, metadata_json`

	for _, bp := range p.Branches {
		b := &models.Branch{}
		err := tx.QueryRow(ctx, branchQ, exp.ID, bp.Key, bp.Name, bp.Weight, nilIfEmpty(bp.MetadataJSON)).Scan(
			&b.ID, &b.ExperimentID, &b.Key, &b.Name, &b.Weight, &b.MetadataJSON,
		)
		if err != nil {
			return nil, classifyBranchErr("create branch in transaction", err)
		}
		exp.Branches = append(exp.Branches, b)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return exp, nil
}

func (s *ExperimentStore) List(ctx context.Context, applicationID string) ([]*models.Experiment, error) {
	const q = `
		SELECT id, application_id, key, name, description, status, start_date, end_date, created_at, updated_at
		FROM experiments
		WHERE application_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, q, applicationID)
	if err != nil {
		return nil, fmt.Errorf("list experiments: %w", err)
	}
	defer rows.Close()

	var exps []*models.Experiment
	for rows.Next() {
		exp := &models.Experiment{Branches: []*models.Branch{}}
		if err := rows.Scan(
			&exp.ID, &exp.ApplicationID, &exp.Key, &exp.Name, &exp.Description,
			&exp.Status, &exp.StartDate, &exp.EndDate, &exp.CreatedAt, &exp.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan experiment: %w", err)
		}
		exps = append(exps, exp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list experiments rows: %w", err)
	}

	return exps, nil
}

func (s *ExperimentStore) GetByID(ctx context.Context, applicationID, id string) (*models.Experiment, error) {
	const q = `
		SELECT id, application_id, key, name, description, status, start_date, end_date, created_at, updated_at
		FROM experiments
		WHERE id = $1 AND application_id = $2 AND deleted_at IS NULL`

	exp := &models.Experiment{Branches: []*models.Branch{}}
	err := s.pool.QueryRow(ctx, q, id, applicationID).Scan(
		&exp.ID, &exp.ApplicationID, &exp.Key, &exp.Name, &exp.Description,
		&exp.Status, &exp.StartDate, &exp.EndDate, &exp.CreatedAt, &exp.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get experiment: %w", err)
	}

	return exp, nil
}

func (s *ExperimentStore) Update(ctx context.Context, applicationID, id string, p UpdateExperimentParams) (*models.Experiment, error) {
	const q = `
		UPDATE experiments
		SET name = $1, description = $2, status = $3, start_date = $4, end_date = $5, updated_at = NOW()
		WHERE id = $6 AND application_id = $7 AND deleted_at IS NULL
		RETURNING id, application_id, key, name, description, status, start_date, end_date, created_at, updated_at`

	exp := &models.Experiment{Branches: []*models.Branch{}}
	err := s.pool.QueryRow(ctx, q,
		p.Name, p.Description, p.Status, p.StartDate, p.EndDate, id, applicationID,
	).Scan(
		&exp.ID, &exp.ApplicationID, &exp.Key, &exp.Name, &exp.Description,
		&exp.Status, &exp.StartDate, &exp.EndDate, &exp.CreatedAt, &exp.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, classifyExperimentErr("update experiment", err)
	}

	return exp, nil
}

func classifyExperimentErr(op string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrConflict
		case "23503":
			return ErrNotFound
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}

func (s *ExperimentStore) Delete(ctx context.Context, applicationID, id string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const softDeleteBranches = `
		UPDATE branches
		SET deleted_at = NOW()
		WHERE experiment_id = $1 AND deleted_at IS NULL`

	if _, err := tx.Exec(ctx, softDeleteBranches, id); err != nil {
		return fmt.Errorf("delete experiment branches: %w", err)
	}

	const softDeleteExperiment = `
		UPDATE experiments
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND application_id = $2 AND deleted_at IS NULL`

	tag, err := tx.Exec(ctx, softDeleteExperiment, id, applicationID)
	if err != nil {
		return fmt.Errorf("delete experiment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// pgxQuerier is satisfied by both *pgxpool.Pool and pgx.Tx.
type pgxQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

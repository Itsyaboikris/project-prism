package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"project-prism/server/internal/models"
)

type CreateBranchParams struct {
	ExperimentID string
	Key          string
	Name         string
	Weight       float64
	MetadataJSON json.RawMessage
}

type UpdateBranchParams struct {
	Name         string
	Weight       float64
	MetadataJSON json.RawMessage
}

type BranchStore struct {
	pool *pgxpool.Pool
}

func NewBranchStore(pool *pgxpool.Pool) *BranchStore {
	return &BranchStore{pool: pool}
}

func (s *BranchStore) Create(ctx context.Context, p CreateBranchParams) (*models.Branch, error) {
	const q = `
		INSERT INTO branches (experiment_id, key, name, weight, metadata_json)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, experiment_id, key, name, weight, metadata_json`

	b := &models.Branch{}
	err := s.pool.QueryRow(ctx, q, p.ExperimentID, p.Key, p.Name, p.Weight, nilIfEmpty(p.MetadataJSON)).Scan(
		&b.ID, &b.ExperimentID, &b.Key, &b.Name, &b.Weight, &b.MetadataJSON,
	)
	if err != nil {
		return nil, classifyBranchErr("create branch", err)
	}

	return b, nil
}

func (s *BranchStore) ListByExperimentID(ctx context.Context, experimentID string) ([]*models.Branch, error) {
	return s.listByIDs(ctx, []string{experimentID})
}

// ListByExperimentIDs fetches branches for multiple experiments in one query.
// Returns a map of experimentID → branches.
func (s *BranchStore) ListByExperimentIDs(ctx context.Context, experimentIDs []string) (map[string][]*models.Branch, error) {
	if len(experimentIDs) == 0 {
		return map[string][]*models.Branch{}, nil
	}

	const q = `
		SELECT id, experiment_id, key, name, weight, metadata_json
		FROM branches
		WHERE experiment_id = ANY($1)
		ORDER BY experiment_id, name`

	rows, err := s.pool.Query(ctx, q, experimentIDs)
	if err != nil {
		return nil, fmt.Errorf("list branches by experiment ids: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]*models.Branch)
	for rows.Next() {
		b := &models.Branch{}
		if err := rows.Scan(&b.ID, &b.ExperimentID, &b.Key, &b.Name, &b.Weight, &b.MetadataJSON); err != nil {
			return nil, fmt.Errorf("scan branch: %w", err)
		}
		result[b.ExperimentID] = append(result[b.ExperimentID], b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list branches rows: %w", err)
	}

	return result, nil
}

func (s *BranchStore) GetByID(ctx context.Context, experimentID, id string) (*models.Branch, error) {
	const q = `
		SELECT id, experiment_id, key, name, weight, metadata_json
		FROM branches
		WHERE id = $1 AND experiment_id = $2`

	b := &models.Branch{}
	err := s.pool.QueryRow(ctx, q, id, experimentID).Scan(
		&b.ID, &b.ExperimentID, &b.Key, &b.Name, &b.Weight, &b.MetadataJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get branch: %w", err)
	}

	return b, nil
}

func (s *BranchStore) Update(ctx context.Context, experimentID, id string, p UpdateBranchParams) (*models.Branch, error) {
	const q = `
		UPDATE branches
		SET name = $1, weight = $2, metadata_json = $3
		WHERE id = $4 AND experiment_id = $5
		RETURNING id, experiment_id, key, name, weight, metadata_json`

	b := &models.Branch{}
	err := s.pool.QueryRow(ctx, q, p.Name, p.Weight, nilIfEmpty(p.MetadataJSON), id, experimentID).Scan(
		&b.ID, &b.ExperimentID, &b.Key, &b.Name, &b.Weight, &b.MetadataJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, classifyBranchErr("update branch", err)
	}

	return b, nil
}

func (s *BranchStore) Delete(ctx context.Context, experimentID, id string) error {
	const q = `DELETE FROM branches WHERE id = $1 AND experiment_id = $2`

	tag, err := s.pool.Exec(ctx, q, id, experimentID)
	if err != nil {
		return fmt.Errorf("delete branch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *BranchStore) listByIDs(ctx context.Context, experimentIDs []string) ([]*models.Branch, error) {
	byExp, err := s.ListByExperimentIDs(ctx, experimentIDs)
	if err != nil {
		return nil, err
	}
	if len(experimentIDs) == 1 {
		if branches, ok := byExp[experimentIDs[0]]; ok {
			return branches, nil
		}
		return []*models.Branch{}, nil
	}
	var all []*models.Branch
	for _, branches := range byExp {
		all = append(all, branches...)
	}
	return all, nil
}

func classifyBranchErr(op string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation — duplicate (experiment_id, key)
			return ErrConflict
		case "23503": // foreign_key_violation — experiment does not exist
			return ErrNotFound
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}

// nilIfEmpty returns nil for an empty json.RawMessage so pgx inserts NULL.
func nilIfEmpty(b json.RawMessage) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

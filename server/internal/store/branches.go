package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	Key          string
	Name         string
	Weight       float64
	MetadataJSON json.RawMessage
}

type SaveBranchParams struct {
	ID           string
	Key          string
	Name         string
	Weight       float64
	MetadataJSON json.RawMessage
}

type BranchStore struct {
	pool DB
}

func NewBranchStore(pool DB) *BranchStore {
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
		WHERE experiment_id = ANY($1) AND deleted_at IS NULL
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
		WHERE id = $1 AND experiment_id = $2 AND deleted_at IS NULL`

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
		WHERE id = $4 AND experiment_id = $5 AND deleted_at IS NULL
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
	const q = `
		UPDATE branches
		SET deleted_at = NOW()
		WHERE id = $1 AND experiment_id = $2 AND deleted_at IS NULL`

	tag, err := s.pool.Exec(ctx, q, id, experimentID)
	if err != nil {
		return fmt.Errorf("delete branch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *BranchStore) SaveAll(ctx context.Context, experimentID string, branches []SaveBranchParams) ([]*models.Branch, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin branch save transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	existingBranches, err := listBranchesByExperimentID(ctx, tx, experimentID)
	if err != nil {
		return nil, err
	}

	existingByID := make(map[string]*models.Branch, len(existingBranches))
	for _, branch := range existingBranches {
		existingByID[branch.ID] = branch
	}

	keptIDs := make(map[string]struct{}, len(branches))
	for _, branch := range branches {
		if branch.ID == "" {
			continue
		}
		if _, ok := existingByID[branch.ID]; !ok {
			return nil, ErrNotFound
		}
		keptIDs[branch.ID] = struct{}{}
	}

	for _, branch := range existingBranches {
		if _, ok := keptIDs[branch.ID]; ok {
			continue
		}
		if err := deleteBranch(ctx, tx, experimentID, branch.ID); err != nil {
			return nil, err
		}
	}

	for _, branch := range branches {
		if branch.ID == "" {
			if _, err := createBranch(ctx, tx, experimentID, CreateBranchParams{
				ExperimentID: experimentID,
				Key:          branch.Key,
				Name:         branch.Name,
				Weight:       branch.Weight,
				MetadataJSON: branch.MetadataJSON,
			}); err != nil {
				return nil, err
			}
			continue
		}

		if _, err := updateBranch(ctx, tx, experimentID, branch.ID, UpdateBranchParams{
			Key:          branch.Key,
			Name:         branch.Name,
			Weight:       branch.Weight,
			MetadataJSON: branch.MetadataJSON,
		}); err != nil {
			return nil, err
		}
	}

	updatedBranches, err := listBranchesByExperimentID(ctx, tx, experimentID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit branch save transaction: %w", err)
	}

	return updatedBranches, nil
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

type branchQueryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func listBranchesByExperimentID(ctx context.Context, q branchQueryer, experimentID string) ([]*models.Branch, error) {
	const query = `
		SELECT id, experiment_id, key, name, weight, metadata_json
		FROM branches
		WHERE experiment_id = $1 AND deleted_at IS NULL
		ORDER BY name`

	rows, err := q.Query(ctx, query, experimentID)
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}
	defer rows.Close()

	var branches []*models.Branch
	for rows.Next() {
		branch := &models.Branch{}
		if err := rows.Scan(
			&branch.ID,
			&branch.ExperimentID,
			&branch.Key,
			&branch.Name,
			&branch.Weight,
			&branch.MetadataJSON,
		); err != nil {
			return nil, fmt.Errorf("scan branch: %w", err)
		}
		branches = append(branches, branch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list branches rows: %w", err)
	}

	return branches, nil
}

func createBranch(ctx context.Context, q branchQueryer, experimentID string, p CreateBranchParams) (*models.Branch, error) {
	const query = `
		INSERT INTO branches (experiment_id, key, name, weight, metadata_json)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, experiment_id, key, name, weight, metadata_json`

	branch := &models.Branch{}
	err := q.QueryRow(ctx, query, experimentID, p.Key, p.Name, p.Weight, nilIfEmpty(p.MetadataJSON)).Scan(
		&branch.ID, &branch.ExperimentID, &branch.Key, &branch.Name, &branch.Weight, &branch.MetadataJSON,
	)
	if err != nil {
		return nil, classifyBranchErr("create branch", err)
	}

	return branch, nil
}

func updateBranch(ctx context.Context, q branchQueryer, experimentID, id string, p UpdateBranchParams) (*models.Branch, error) {
	const query = `
		UPDATE branches
		SET key = $1, name = $2, weight = $3, metadata_json = $4
		WHERE id = $5 AND experiment_id = $6 AND deleted_at IS NULL
		RETURNING id, experiment_id, key, name, weight, metadata_json`

	branch := &models.Branch{}
	err := q.QueryRow(ctx, query, p.Key, p.Name, p.Weight, nilIfEmpty(p.MetadataJSON), id, experimentID).Scan(
		&branch.ID, &branch.ExperimentID, &branch.Key, &branch.Name, &branch.Weight, &branch.MetadataJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, classifyBranchErr("update branch", err)
	}

	return branch, nil
}

func deleteBranch(ctx context.Context, q branchQueryer, experimentID, id string) error {
	const query = `
		UPDATE branches
		SET deleted_at = NOW()
		WHERE id = $1 AND experiment_id = $2 AND deleted_at IS NULL`

	tag, err := q.Exec(ctx, query, id, experimentID)
	if err != nil {
		return fmt.Errorf("delete branch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// nilIfEmpty returns nil for an empty json.RawMessage so pgx inserts NULL.
func nilIfEmpty(b json.RawMessage) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

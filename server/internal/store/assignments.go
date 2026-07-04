package store

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"project-prism/server/internal/models"
)

var ErrNotEligible = errors.New("record not eligible")
var ErrMisconfigured = errors.New("record misconfigured")

type AssignParams struct {
	ApplicationID string
	ExperimentKey string
	UserID        string
}

type AssignmentStore struct {
	pool DB
}

func NewAssignmentStore(pool DB) *AssignmentStore {
	return &AssignmentStore{pool: pool}
}

func (s *AssignmentStore) Assign(ctx context.Context, p AssignParams) (*models.Branch, error) {
	experiment, err := s.getExperimentForAssignment(ctx, p.ApplicationID, p.ExperimentKey)
	if err != nil {
		return nil, err
	}

	branch, err := s.getExistingAssignedBranch(ctx, experiment.ID, p.UserID)
	if err != nil {
		return nil, err
	}
	if branch != nil {
		return branch, nil
	}

	branches, err := s.listActiveBranches(ctx, experiment.ID)
	if err != nil {
		return nil, err
	}
	if len(branches) == 0 {
		return nil, ErrMisconfigured
	}

	branch, err = selectBranch(p.ApplicationID, p.ExperimentKey, p.UserID, branches)
	if err != nil {
		return nil, err
	}

	branchID, err := s.upsertAssignment(ctx, p.ApplicationID, experiment.ID, branch.ID, p.UserID)
	if err != nil {
		return nil, err
	}

	return s.getBranchByID(ctx, experiment.ID, branchID)
}

type assignmentExperiment struct {
	ID        string
	Status    models.ExperimentStatus
	StartDate *time.Time
	EndDate   *time.Time
}

func (s *AssignmentStore) getExperimentForAssignment(ctx context.Context, applicationID, experimentKey string) (*assignmentExperiment, error) {
	const q = `
		SELECT id, status, start_date, end_date
		FROM experiments
		WHERE application_id = $1 AND key = $2 AND deleted_at IS NULL`

	experiment := &assignmentExperiment{}
	err := s.pool.QueryRow(ctx, q, applicationID, experimentKey).Scan(
		&experiment.ID,
		&experiment.Status,
		&experiment.StartDate,
		&experiment.EndDate,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get experiment for assignment: %w", err)
	}

	now := time.Now()
	if experiment.Status != models.ExperimentStatusActive {
		return nil, ErrNotEligible
	}
	if experiment.StartDate != nil && experiment.StartDate.After(now) {
		return nil, ErrNotEligible
	}
	if experiment.EndDate != nil && experiment.EndDate.Before(now) {
		return nil, ErrNotEligible
	}

	return experiment, nil
}

func (s *AssignmentStore) getExistingAssignedBranch(ctx context.Context, experimentID, userID string) (*models.Branch, error) {
	const q = `
		SELECT b.id, b.experiment_id, b.key, b.name, b.weight, b.metadata_json
		FROM assignments a
		JOIN branches b ON b.id = a.branch_id
		WHERE a.experiment_id = $1
		  AND a.user_id = $2
		  AND b.deleted_at IS NULL`

	branch := &models.Branch{}
	err := s.pool.QueryRow(ctx, q, experimentID, userID).Scan(
		&branch.ID,
		&branch.ExperimentID,
		&branch.Key,
		&branch.Name,
		&branch.Weight,
		&branch.MetadataJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get existing assignment: %w", err)
	}

	return branch, nil
}

func (s *AssignmentStore) listActiveBranches(ctx context.Context, experimentID string) ([]*models.Branch, error) {
	const q = `
		SELECT id, experiment_id, key, name, weight, metadata_json
		FROM branches
		WHERE experiment_id = $1 AND deleted_at IS NULL
		ORDER BY key`

	rows, err := s.pool.Query(ctx, q, experimentID)
	if err != nil {
		return nil, fmt.Errorf("list branches for assignment: %w", err)
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
			return nil, fmt.Errorf("scan assignment branch: %w", err)
		}
		branches = append(branches, branch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list assignment branches rows: %w", err)
	}

	return branches, nil
}

func (s *AssignmentStore) upsertAssignment(ctx context.Context, applicationID, experimentID, branchID, userID string) (string, error) {
	const q = `
		INSERT INTO assignments (application_id, experiment_id, branch_id, user_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (experiment_id, user_id)
		DO UPDATE SET
			application_id = EXCLUDED.application_id,
			branch_id = EXCLUDED.branch_id,
			updated_at = NOW()
		RETURNING branch_id`

	var assignedBranchID string
	if err := s.pool.QueryRow(ctx, q, applicationID, experimentID, branchID, userID).Scan(&assignedBranchID); err != nil {
		return "", fmt.Errorf("upsert assignment: %w", err)
	}

	return assignedBranchID, nil
}

func (s *AssignmentStore) getBranchByID(ctx context.Context, experimentID, branchID string) (*models.Branch, error) {
	const q = `
		SELECT id, experiment_id, key, name, weight, metadata_json
		FROM branches
		WHERE id = $1 AND experiment_id = $2 AND deleted_at IS NULL`

	branch := &models.Branch{}
	err := s.pool.QueryRow(ctx, q, branchID, experimentID).Scan(
		&branch.ID,
		&branch.ExperimentID,
		&branch.Key,
		&branch.Name,
		&branch.Weight,
		&branch.MetadataJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get assigned branch: %w", err)
	}

	return branch, nil
}

func selectBranch(applicationID, experimentKey, userID string, branches []*models.Branch) (*models.Branch, error) {
	totalWeight := 0.0
	for _, branch := range branches {
		totalWeight += branch.Weight
	}
	if totalWeight <= 0 {
		return nil, ErrMisconfigured
	}

	// Branch weights are stored at 4 decimal places, so a 10,000 bucket hash
	// gives stable deterministic assignment without requiring randomness.
	sum := sha256.Sum256([]byte(applicationID + ":" + experimentKey + ":" + userID))
	bucket := binary.BigEndian.Uint64(sum[:8]) % 10000
	target := float64(bucket) / 10000.0

	cumulative := 0.0
	for i, branch := range branches {
		cumulative += branch.Weight / totalWeight
		if target < cumulative || i == len(branches)-1 {
			return branch, nil
		}
	}

	return nil, ErrMisconfigured
}

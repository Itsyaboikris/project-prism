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

type assignmentExperimentView struct {
	ID     string
	Key    string
	Name   string
	Status models.ExperimentStatus
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

func (s *AssignmentStore) ListByExperiment(ctx context.Context, applicationID, experimentID string) (*models.ExperimentAssignmentsView, error) {
	experiment, err := s.getExperimentView(ctx, applicationID, experimentID)
	if err != nil {
		return nil, err
	}

	const q = `
		SELECT a.id, a.application_id, a.experiment_id, a.branch_id, a.user_id, a.assigned_at,
		       a.context_json, a.created_at, a.updated_at, b.key, b.name, b.weight
		FROM assignments a
		JOIN branches b ON b.id = a.branch_id
		WHERE a.application_id = $1
		  AND a.experiment_id = $2
		ORDER BY a.assigned_at DESC, a.id DESC`

	rows, err := s.pool.Query(ctx, q, applicationID, experimentID)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	defer rows.Close()

	view := &models.ExperimentAssignmentsView{
		ExperimentID:     experiment.ID,
		ExperimentKey:    experiment.Key,
		ExperimentName:   experiment.Name,
		ExperimentStatus: experiment.Status,
		Assignments:      []*models.ExperimentAssignmentListItem{},
	}

	for rows.Next() {
		assignment := &models.ExperimentAssignmentListItem{}
		if err := rows.Scan(
			&assignment.ID,
			&assignment.ApplicationID,
			&assignment.ExperimentID,
			&assignment.BranchID,
			&assignment.UserID,
			&assignment.AssignedAt,
			&assignment.ContextJSON,
			&assignment.CreatedAt,
			&assignment.UpdatedAt,
			&assignment.BranchKey,
			&assignment.BranchName,
			&assignment.BranchWeight,
		); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		view.Assignments = append(view.Assignments, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list assignments rows: %w", err)
	}

	return view, nil
}

func (s *AssignmentStore) GetExperimentDashboard(ctx context.Context, applicationID, experimentID string) (*models.ExperimentDashboard, error) {
	experiment, err := s.getExperimentView(ctx, applicationID, experimentID)
	if err != nil {
		return nil, err
	}

	const q = `
		SELECT b.id, b.key, b.name, b.weight, COUNT(a.id)::bigint AS assignment_count
		FROM branches b
		LEFT JOIN assignments a
		  ON a.branch_id = b.id
		 AND a.experiment_id = $1
		WHERE b.experiment_id = $1
		  AND b.deleted_at IS NULL
		GROUP BY b.id, b.key, b.name, b.weight
		ORDER BY b.key`

	rows, err := s.pool.Query(ctx, q, experimentID)
	if err != nil {
		return nil, fmt.Errorf("get experiment dashboard: %w", err)
	}
	defer rows.Close()

	dashboard := &models.ExperimentDashboard{
		ExperimentID:     experiment.ID,
		ExperimentKey:    experiment.Key,
		ExperimentName:   experiment.Name,
		ExperimentStatus: experiment.Status,
		Branches:         []*models.ExperimentDashboardBranch{},
	}

	totalAssignments := 0
	for rows.Next() {
		branch := &models.ExperimentDashboardBranch{}
		var assignmentCount int64
		if err := rows.Scan(
			&branch.BranchID,
			&branch.BranchKey,
			&branch.BranchName,
			&branch.ConfiguredWeight,
			&assignmentCount,
		); err != nil {
			return nil, fmt.Errorf("scan experiment dashboard branch: %w", err)
		}
		branch.AssignmentCount = int(assignmentCount)
		totalAssignments += branch.AssignmentCount
		dashboard.Branches = append(dashboard.Branches, branch)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("experiment dashboard rows: %w", err)
	}

	dashboard.TotalAssignments = totalAssignments
	dashboard.BranchCount = len(dashboard.Branches)
	if dashboard.TotalAssignments > 0 {
		for _, branch := range dashboard.Branches {
			branch.AssignmentShare = (float64(branch.AssignmentCount) / float64(dashboard.TotalAssignments)) * 100
		}
	}

	return dashboard, nil
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

func (s *AssignmentStore) getExperimentView(ctx context.Context, applicationID, experimentID string) (*assignmentExperimentView, error) {
	const q = `
		SELECT id, key, name, status
		FROM experiments
		WHERE application_id = $1
		  AND id = $2
		  AND deleted_at IS NULL`

	experiment := &assignmentExperimentView{}
	err := s.pool.QueryRow(ctx, q, applicationID, experimentID).Scan(
		&experiment.ID,
		&experiment.Key,
		&experiment.Name,
		&experiment.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get experiment view: %w", err)
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

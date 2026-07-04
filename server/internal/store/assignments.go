package store

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
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

type assignmentQueryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
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
	experiment, err := getExperimentForAssignment(ctx, s.pool, p.ApplicationID, p.ExperimentKey, false)
	if err != nil {
		return nil, err
	}

	branch, err := getExistingAssignedBranch(ctx, s.pool, experiment.ID, p.UserID)
	if err != nil {
		return nil, err
	}
	if branch != nil {
		return branch, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin assignment transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	experiment, err = getExperimentForAssignment(ctx, tx, p.ApplicationID, p.ExperimentKey, true)
	if err != nil {
		return nil, err
	}

	branch, err = getExistingAssignedBranch(ctx, tx, experiment.ID, p.UserID)
	if err != nil {
		return nil, err
	}
	if branch != nil {
		return branch, nil
	}

	branches, err := listActiveBranches(ctx, tx, experiment.ID)
	if err != nil {
		return nil, err
	}
	if len(branches) == 0 {
		return nil, ErrMisconfigured
	}

	assignmentCounts, err := getAssignmentCountsByBranch(ctx, tx, experiment.ID)
	if err != nil {
		return nil, err
	}

	branch, err = selectBalancedBranch(p.ApplicationID, p.ExperimentKey, p.UserID, branches, assignmentCounts)
	if err != nil {
		return nil, err
	}

	branchID, err := upsertAssignment(ctx, tx, p.ApplicationID, experiment.ID, branch.ID, p.UserID)
	if err != nil {
		return nil, err
	}

	branch, err = getBranchByID(ctx, tx, experiment.ID, branchID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit assignment transaction: %w", err)
	}

	return branch, nil
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

func getExperimentForAssignment(ctx context.Context, q assignmentQueryer, applicationID, experimentKey string, lock bool) (*assignmentExperiment, error) {
	query := `
		SELECT id, status, start_date, end_date
		FROM experiments
		WHERE application_id = $1 AND key = $2 AND deleted_at IS NULL`
	if lock {
		query += ` FOR UPDATE`
	}

	experiment := &assignmentExperiment{}
	err := q.QueryRow(ctx, query, applicationID, experimentKey).Scan(
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

func getExistingAssignedBranch(ctx context.Context, queryer assignmentQueryer, experimentID, userID string) (*models.Branch, error) {
	const query = `
		SELECT b.id, b.experiment_id, b.key, b.name, b.weight, b.metadata_json
		FROM assignments a
		JOIN branches b ON b.id = a.branch_id
		WHERE a.experiment_id = $1
		  AND a.user_id = $2
		  AND b.deleted_at IS NULL`

	branch := &models.Branch{}
	err := queryer.QueryRow(ctx, query, experimentID, userID).Scan(
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

func listActiveBranches(ctx context.Context, queryer assignmentQueryer, experimentID string) ([]*models.Branch, error) {
	const query = `
		SELECT id, experiment_id, key, name, weight, metadata_json
		FROM branches
		WHERE experiment_id = $1 AND deleted_at IS NULL
		ORDER BY key`

	rows, err := queryer.Query(ctx, query, experimentID)
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

func getAssignmentCountsByBranch(ctx context.Context, queryer assignmentQueryer, experimentID string) (map[string]int, error) {
	const query = `
		SELECT branch_id, COUNT(*)::bigint AS assignment_count
		FROM assignments
		WHERE experiment_id = $1
		GROUP BY branch_id`

	rows, err := queryer.Query(ctx, query, experimentID)
	if err != nil {
		return nil, fmt.Errorf("get assignment counts by branch: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var branchID string
		var assignmentCount int64
		if err := rows.Scan(&branchID, &assignmentCount); err != nil {
			return nil, fmt.Errorf("scan assignment count: %w", err)
		}
		counts[branchID] = int(assignmentCount)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("assignment counts rows: %w", err)
	}

	return counts, nil
}

func upsertAssignment(ctx context.Context, queryer assignmentQueryer, applicationID, experimentID, branchID, userID string) (string, error) {
	const query = `
		INSERT INTO assignments (application_id, experiment_id, branch_id, user_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (experiment_id, user_id)
		DO UPDATE SET
			application_id = EXCLUDED.application_id,
			branch_id = EXCLUDED.branch_id,
			updated_at = NOW()
		RETURNING branch_id`

	var assignedBranchID string
	if err := queryer.QueryRow(ctx, query, applicationID, experimentID, branchID, userID).Scan(&assignedBranchID); err != nil {
		return "", fmt.Errorf("upsert assignment: %w", err)
	}

	return assignedBranchID, nil
}

func getBranchByID(ctx context.Context, queryer assignmentQueryer, experimentID, branchID string) (*models.Branch, error) {
	const query = `
		SELECT id, experiment_id, key, name, weight, metadata_json
		FROM branches
		WHERE id = $1 AND experiment_id = $2 AND deleted_at IS NULL`

	branch := &models.Branch{}
	err := queryer.QueryRow(ctx, query, branchID, experimentID).Scan(
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

func selectBalancedBranch(
	applicationID, experimentKey, userID string,
	branches []*models.Branch,
	assignmentCounts map[string]int,
) (*models.Branch, error) {
	totalWeight := 0.0
	for _, branch := range branches {
		totalWeight += branch.Weight
	}
	if totalWeight <= 0 {
		return nil, ErrMisconfigured
	}

	totalAssignments := 0
	for _, branch := range branches {
		totalAssignments += assignmentCounts[branch.ID]
	}

	var selected *models.Branch
	bestScore := 0.0
	bestTieBreaker := uint64(0)
	const scoreTolerance = 0.0000001

	for _, branch := range branches {
		currentCount := assignmentCounts[branch.ID]
		score := (branch.Weight * float64(totalAssignments+1)) - (float64(currentCount) * totalWeight)
		tieBreaker := assignmentTieBucket(applicationID, experimentKey, userID, branch.ID)

		if selected == nil ||
			score > bestScore+scoreTolerance ||
			(math.Abs(score-bestScore) <= scoreTolerance && tieBreaker < bestTieBreaker) {
			selected = branch
			bestScore = score
			bestTieBreaker = tieBreaker
		}
	}

	if selected == nil {
		return nil, ErrMisconfigured
	}

	return selected, nil
}

func assignmentTieBucket(applicationID, experimentKey, userID, branchID string) uint64 {
	sum := sha256.Sum256([]byte(applicationID + ":" + experimentKey + ":" + userID + ":" + branchID))
	return binary.BigEndian.Uint64(sum[:8])
}

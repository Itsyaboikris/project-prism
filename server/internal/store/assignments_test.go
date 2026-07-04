package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	"project-prism/server/internal/models"
)

func TestSelectBalancedBranchBalancesEvenWeights(t *testing.T) {
	branches := []*models.Branch{
		{ID: "branch_control", Key: "control", Weight: 50},
		{ID: "branch_variant", Key: "variant-a", Weight: 50},
	}
	counts := map[string]int{}

	for i := range 10 {
		branch, err := selectBalancedBranch(
			"app_123",
			"checkout-button-color",
			fmt.Sprintf("user_%d", i),
			branches,
			counts,
		)
		if err != nil {
			t.Fatalf("selectBalancedBranch returned error: %v", err)
		}
		counts[branch.ID]++
		if diff := math.Abs(float64(counts["branch_control"] - counts["branch_variant"])); diff > 1 {
			t.Fatalf("expected counts to stay closely balanced, got %#v", counts)
		}
	}

	if counts["branch_control"] != 5 || counts["branch_variant"] != 5 {
		t.Fatalf("expected final 50/50 split, got %#v", counts)
	}
}

func TestSelectBalancedBranchHonorsWeights(t *testing.T) {
	branches := []*models.Branch{
		{ID: "branch_control", Key: "control", Weight: 80},
		{ID: "branch_variant", Key: "variant-a", Weight: 20},
	}
	counts := map[string]int{}

	for i := range 10 {
		branch, err := selectBalancedBranch(
			"app_123",
			"checkout-button-color",
			fmt.Sprintf("user_%d", i),
			branches,
			counts,
		)
		if err != nil {
			t.Fatalf("selectBalancedBranch returned error: %v", err)
		}
		counts[branch.ID]++
	}

	if counts["branch_control"] != 8 || counts["branch_variant"] != 2 {
		t.Fatalf("expected final 80/20 split after 10 users, got %#v", counts)
	}
}

func TestSelectBalancedBranchTieBreaksDeterministically(t *testing.T) {
	branches := []*models.Branch{
		{ID: "branch_control", Key: "control", Weight: 50},
		{ID: "branch_variant", Key: "variant-a", Weight: 50},
	}
	counts := map[string]int{
		"branch_control": 3,
		"branch_variant": 3,
	}

	first, err := selectBalancedBranch("app_123", "checkout-button-color", "user_123", branches, counts)
	if err != nil {
		t.Fatalf("first selectBalancedBranch: %v", err)
	}

	second, err := selectBalancedBranch("app_123", "checkout-button-color", "user_123", branches, counts)
	if err != nil {
		t.Fatalf("second selectBalancedBranch: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("expected deterministic tie-break, got %q then %q", first.ID, second.ID)
	}
}

func TestSelectBalancedBranchRejectsMisconfiguredWeights(t *testing.T) {
	branches := []*models.Branch{
		{ID: "branch_a", Key: "a", Weight: 0},
		{ID: "branch_b", Key: "b", Weight: 0},
	}

	_, err := selectBalancedBranch("app_123", "checkout-button-color", "user_123", branches, map[string]int{})
	if err != ErrMisconfigured {
		t.Fatalf("expected ErrMisconfigured, got %v", err)
	}
}

func TestAssignmentStoreAssignKeepsExistingAssignmentSticky(t *testing.T) {
	mock := newMockPool(t)
	store := NewAssignmentStore(mock)

	experimentRows := pgxmock.NewRows([]string{"id", "status", "start_date", "end_date"}).
		AddRow("exp_123", "active", nil, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, status, start_date, end_date
		FROM experiments
		WHERE application_id = $1 AND key = $2 AND deleted_at IS NULL`)).
		WithArgs("app_123", "checkout-button-color").
		WillReturnRows(experimentRows)

	existingAssignmentRows := pgxmock.NewRows([]string{"id", "experiment_id", "key", "name", "weight", "metadata_json"}).
		AddRow("branch_123", "exp_123", "control", "Control", 50.0, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT b.id, b.experiment_id, b.key, b.name, b.weight, b.metadata_json
		FROM assignments a
		JOIN branches b ON b.id = a.branch_id
		WHERE a.experiment_id = $1
		  AND a.user_id = $2
		  AND b.deleted_at IS NULL`)).
		WithArgs("exp_123", "user_123").
		WillReturnRows(existingAssignmentRows)

	got, err := store.Assign(context.Background(), AssignParams{
		ApplicationID: "app_123",
		ExperimentKey: "checkout-button-color",
		UserID:        "user_123",
	})
	if err != nil {
		t.Fatalf("Assign returned error: %v", err)
	}
	if got.ID != "branch_123" {
		t.Fatalf("expected sticky branch branch_123, got %#v", got)
	}
}

func TestAssignmentStoreAssignUsesCountAwareTransactionForNewUsers(t *testing.T) {
	mock := newMockPool(t)
	store := NewAssignmentStore(mock)

	experimentRows := pgxmock.NewRows([]string{"id", "status", "start_date", "end_date"}).
		AddRow("exp_123", "active", nil, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, status, start_date, end_date
		FROM experiments
		WHERE application_id = $1 AND key = $2 AND deleted_at IS NULL`)).
		WithArgs("app_123", "checkout-button-color").
		WillReturnRows(experimentRows)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT b.id, b.experiment_id, b.key, b.name, b.weight, b.metadata_json
		FROM assignments a
		JOIN branches b ON b.id = a.branch_id
		WHERE a.experiment_id = $1
		  AND a.user_id = $2
		  AND b.deleted_at IS NULL`)).
		WithArgs("exp_123", "user_123").
		WillReturnError(pgx.ErrNoRows)

	mock.ExpectBegin()

	lockedExperimentRows := pgxmock.NewRows([]string{"id", "status", "start_date", "end_date"}).
		AddRow("exp_123", "active", nil, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, status, start_date, end_date
		FROM experiments
		WHERE application_id = $1 AND key = $2 AND deleted_at IS NULL FOR UPDATE`)).
		WithArgs("app_123", "checkout-button-color").
		WillReturnRows(lockedExperimentRows)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT b.id, b.experiment_id, b.key, b.name, b.weight, b.metadata_json
		FROM assignments a
		JOIN branches b ON b.id = a.branch_id
		WHERE a.experiment_id = $1
		  AND a.user_id = $2
		  AND b.deleted_at IS NULL`)).
		WithArgs("exp_123", "user_123").
		WillReturnError(pgx.ErrNoRows)

	branchRows := pgxmock.NewRows([]string{"id", "experiment_id", "key", "name", "weight", "metadata_json"}).
		AddRow("branch_123", "exp_123", "control", "Control", 50.0, nil).
		AddRow("branch_456", "exp_123", "variant", "Variant", 50.0, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, experiment_id, key, name, weight, metadata_json
		FROM branches
		WHERE experiment_id = $1 AND deleted_at IS NULL
		ORDER BY key`)).
		WithArgs("exp_123").
		WillReturnRows(branchRows)

	countRows := pgxmock.NewRows([]string{"branch_id", "assignment_count"}).
		AddRow("branch_123", int64(1)).
		AddRow("branch_456", int64(0))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT branch_id, COUNT(*)::bigint AS assignment_count
		FROM assignments
		WHERE experiment_id = $1
		GROUP BY branch_id`)).
		WithArgs("exp_123").
		WillReturnRows(countRows)

	upsertRows := pgxmock.NewRows([]string{"branch_id"}).AddRow("branch_456")
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO assignments (application_id, experiment_id, branch_id, user_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (experiment_id, user_id)
		DO UPDATE SET
			application_id = EXCLUDED.application_id,
			branch_id = EXCLUDED.branch_id,
			updated_at = NOW()
		RETURNING branch_id`)).
		WithArgs("app_123", "exp_123", "branch_456", "user_123").
		WillReturnRows(upsertRows)

	assignedBranchRows := pgxmock.NewRows([]string{"id", "experiment_id", "key", "name", "weight", "metadata_json"}).
		AddRow("branch_456", "exp_123", "variant", "Variant", 50.0, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, experiment_id, key, name, weight, metadata_json
		FROM branches
		WHERE id = $1 AND experiment_id = $2 AND deleted_at IS NULL`)).
		WithArgs("branch_456", "exp_123").
		WillReturnRows(assignedBranchRows)

	mock.ExpectCommit()

	got, err := store.Assign(context.Background(), AssignParams{
		ApplicationID: "app_123",
		ExperimentKey: "checkout-button-color",
		UserID:        "user_123",
	})
	if err != nil {
		t.Fatalf("Assign returned error: %v", err)
	}
	if got.ID != "branch_456" {
		t.Fatalf("expected under-target branch branch_456, got %#v", got)
	}
}

func TestAssignmentStoreListByExperiment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockPool(t)
		store := NewAssignmentStore(mock)

		experimentRows := pgxmock.NewRows([]string{"id", "key", "name", "status"}).
			AddRow("exp_123", "checkout-button-color", "Checkout Button Color", "active")
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, key, name, status
		FROM experiments
		WHERE application_id = $1
		  AND id = $2
		  AND deleted_at IS NULL`)).
			WithArgs("app_123", "exp_123").
			WillReturnRows(experimentRows)

		assignmentRows := pgxmock.NewRows([]string{
			"id", "application_id", "experiment_id", "branch_id", "user_id", "assigned_at",
			"context_json", "created_at", "updated_at", "key", "name", "weight",
		}).
			AddRow("assign_456", "app_123", "exp_123", "branch_456", "user_2", nowTime(), nil, nowTime(), nowTime(), "variant", "Variant", 50.0).
			AddRow("assign_123", "app_123", "exp_123", "branch_123", "user_1", nowTime(), nil, nowTime(), nowTime(), "control", "Control", 50.0)
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT a.id, a.application_id, a.experiment_id, a.branch_id, a.user_id, a.assigned_at,
		       a.context_json, a.created_at, a.updated_at, b.key, b.name, b.weight
		FROM assignments a
		JOIN branches b ON b.id = a.branch_id
		WHERE a.application_id = $1
		  AND a.experiment_id = $2
		ORDER BY a.assigned_at DESC, a.id DESC`)).
			WithArgs("app_123", "exp_123").
			WillReturnRows(assignmentRows)

		got, err := store.ListByExperiment(context.Background(), "app_123", "exp_123")
		if err != nil {
			t.Fatalf("ListByExperiment returned error: %v", err)
		}
		if got.ExperimentID != "exp_123" || got.ExperimentKey != "checkout-button-color" {
			t.Fatalf("unexpected experiment view: %#v", got)
		}
		if len(got.Assignments) != 2 || got.Assignments[0].ID != "assign_456" {
			t.Fatalf("unexpected assignments: %#v", got.Assignments)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := newMockPool(t)
		store := NewAssignmentStore(mock)

		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, key, name, status
		FROM experiments
		WHERE application_id = $1
		  AND id = $2
		  AND deleted_at IS NULL`)).
			WithArgs("app_123", "exp_123").
			WillReturnError(pgx.ErrNoRows)

		_, err := store.ListByExperiment(context.Background(), "app_123", "exp_123")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestAssignmentStoreGetExperimentDashboard(t *testing.T) {
	mock := newMockPool(t)
	store := NewAssignmentStore(mock)

	experimentRows := pgxmock.NewRows([]string{"id", "key", "name", "status"}).
		AddRow("exp_123", "checkout-button-color", "Checkout Button Color", "active")
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, key, name, status
		FROM experiments
		WHERE application_id = $1
		  AND id = $2
		  AND deleted_at IS NULL`)).
		WithArgs("app_123", "exp_123").
		WillReturnRows(experimentRows)

	dashboardRows := pgxmock.NewRows([]string{"id", "key", "name", "weight", "assignment_count"}).
		AddRow("branch_123", "control", "Control", 60.0, int64(3)).
		AddRow("branch_456", "variant", "Variant", 40.0, int64(1)).
		AddRow("branch_789", "holdout", "Holdout", 0.0, int64(0))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT b.id, b.key, b.name, b.weight, COUNT(a.id)::bigint AS assignment_count
		FROM branches b
		LEFT JOIN assignments a
		  ON a.branch_id = b.id
		 AND a.experiment_id = $1
		WHERE b.experiment_id = $1
		  AND b.deleted_at IS NULL
		GROUP BY b.id, b.key, b.name, b.weight
		ORDER BY b.key`)).
		WithArgs("exp_123").
		WillReturnRows(dashboardRows)

	got, err := store.GetExperimentDashboard(context.Background(), "app_123", "exp_123")
	if err != nil {
		t.Fatalf("GetExperimentDashboard returned error: %v", err)
	}
	if got.TotalAssignments != 4 || got.BranchCount != 3 {
		t.Fatalf("unexpected dashboard summary: %#v", got)
	}
	if got.Branches[0].BranchKey != "control" || got.Branches[0].AssignmentShare != 75 {
		t.Fatalf("unexpected first branch summary: %#v", got.Branches[0])
	}
	if got.Branches[2].BranchKey != "holdout" || got.Branches[2].AssignmentCount != 0 {
		t.Fatalf("expected zero-assignment branch in dashboard, got %#v", got.Branches[2])
	}
}

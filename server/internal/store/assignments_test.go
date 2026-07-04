package store

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	"project-prism/server/internal/models"
)

func TestSelectBranchIsDeterministic(t *testing.T) {
	branches := []*models.Branch{
		{ID: "branch_control", Key: "control", Weight: 20},
		{ID: "branch_variant", Key: "variant-a", Weight: 80},
	}

	first, err := selectBranch("app_123", "checkout-button-color", "user_123", branches)
	if err != nil {
		t.Fatalf("first selectBranch: %v", err)
	}

	second, err := selectBranch("app_123", "checkout-button-color", "user_123", branches)
	if err != nil {
		t.Fatalf("second selectBranch: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("expected deterministic branch, got %q then %q", first.ID, second.ID)
	}
}

func TestSelectBranchHonorsWeights(t *testing.T) {
	branches := []*models.Branch{
		{ID: "branch_control", Key: "control", Weight: 20},
		{ID: "branch_variant", Key: "variant-a", Weight: 80},
	}

	controlUser := findUserIDForBucketRange(t, "app_123", "checkout-button-color", 0, 1999)
	variantUser := findUserIDForBucketRange(t, "app_123", "checkout-button-color", 2000, 9999)

	controlBranch, err := selectBranch("app_123", "checkout-button-color", controlUser, branches)
	if err != nil {
		t.Fatalf("select control branch: %v", err)
	}
	if controlBranch.Key != "control" {
		t.Fatalf("expected control branch, got %q", controlBranch.Key)
	}

	variantBranch, err := selectBranch("app_123", "checkout-button-color", variantUser, branches)
	if err != nil {
		t.Fatalf("select variant branch: %v", err)
	}
	if variantBranch.Key != "variant-a" {
		t.Fatalf("expected variant branch, got %q", variantBranch.Key)
	}
}

func TestSelectBranchRejectsMisconfiguredWeights(t *testing.T) {
	branches := []*models.Branch{
		{ID: "branch_a", Key: "a", Weight: 0},
		{ID: "branch_b", Key: "b", Weight: 0},
	}

	_, err := selectBranch("app_123", "checkout-button-color", "user_123", branches)
	if err != ErrMisconfigured {
		t.Fatalf("expected ErrMisconfigured, got %v", err)
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

func findUserIDForBucketRange(t *testing.T, applicationID, experimentKey string, min, max uint64) string {
	t.Helper()

	for i := range 100_000 {
		userID := fmt.Sprintf("user_%d", i)
		bucket := assignmentBucket(applicationID, experimentKey, userID)
		if bucket >= min && bucket <= max {
			return userID
		}
	}

	t.Fatalf("no user id found for bucket range %d-%d", min, max)
	return ""
}

func assignmentBucket(applicationID, experimentKey, userID string) uint64 {
	sum := sha256.Sum256([]byte(applicationID + ":" + experimentKey + ":" + userID))
	return binary.BigEndian.Uint64(sum[:8]) % 10000
}

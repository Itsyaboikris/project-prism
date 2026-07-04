package store

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	pgxmock "github.com/pashagolub/pgxmock/v4"
	"project-prism/server/internal/models"
)

func TestExperimentStoreEnsureApplicationActive(t *testing.T) {
	t.Run("active", func(t *testing.T) {
		mock := newMockPool(t)
		store := NewExperimentStore(mock)

		rows := pgxmock.NewRows([]string{"status"}).AddRow("active")
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT status
		FROM applications
		WHERE id = $1 AND deleted_at IS NULL`)).
			WithArgs("app_123").
			WillReturnRows(rows)

		if err := store.ensureApplicationActive(context.Background(), mock, "app_123"); err != nil {
			t.Fatalf("ensureApplicationActive returned error: %v", err)
		}
	})

	t.Run("inactive", func(t *testing.T) {
		mock := newMockPool(t)
		store := NewExperimentStore(mock)

		rows := pgxmock.NewRows([]string{"status"}).AddRow("inactive")
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT status
		FROM applications
		WHERE id = $1 AND deleted_at IS NULL`)).
			WithArgs("app_123").
			WillReturnRows(rows)

		err := store.ensureApplicationActive(context.Background(), mock, "app_123")
		if !errors.Is(err, ErrInactive) {
			t.Fatalf("expected ErrInactive, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := newMockPool(t)
		store := NewExperimentStore(mock)

		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT status
		FROM applications
		WHERE id = $1 AND deleted_at IS NULL`)).
			WithArgs("app_123").
			WillReturnError(pgx.ErrNoRows)

		err := store.ensureApplicationActive(context.Background(), mock, "app_123")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestExperimentStoreCreateAndList(t *testing.T) {
	mock := newMockPool(t)
	store := NewExperimentStore(mock)

	statusRows := pgxmock.NewRows([]string{"status"}).AddRow("active")
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT status
		FROM applications
		WHERE id = $1 AND deleted_at IS NULL`)).
		WithArgs("app_123").
		WillReturnRows(statusRows)

	createRows := pgxmock.NewRows([]string{
		"id", "application_id", "key", "name", "description", "status", "start_date", "end_date", "created_at", "updated_at",
	}).AddRow("exp_123", "app_123", "exp-key", "Experiment", nil, "draft", nil, nil, nowTime(), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO experiments (application_id, key, name, description, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, application_id, key, name, description, status, start_date, end_date, created_at, updated_at`)).
		WithArgs("app_123", "exp-key", "Experiment", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(createRows)

	listRows := pgxmock.NewRows([]string{
		"id", "application_id", "key", "name", "description", "status", "start_date", "end_date", "created_at", "updated_at",
	}).AddRow("exp_123", "app_123", "exp-key", "Experiment", nil, "draft", nil, nil, nowTime(), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, application_id, key, name, description, status, start_date, end_date, created_at, updated_at
		FROM experiments
		WHERE application_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`)).
		WithArgs("app_123").
		WillReturnRows(listRows)

	if _, err := store.Create(context.Background(), CreateExperimentParams{
		ApplicationID: "app_123",
		Key:           "exp-key",
		Name:          "Experiment",
	}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	experiments, err := store.List(context.Background(), "app_123")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(experiments) != 1 {
		t.Fatalf("expected 1 experiment, got %d", len(experiments))
	}
}

func TestExperimentStoreCreateWithBranches(t *testing.T) {
	mock := newMockPool(t)
	store := NewExperimentStore(mock)

	statusRows := pgxmock.NewRows([]string{"status"}).AddRow("active")
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT status
		FROM applications
		WHERE id = $1 AND deleted_at IS NULL`)).
		WithArgs("app_123").
		WillReturnRows(statusRows)

	mock.ExpectBegin()
	createRows := pgxmock.NewRows([]string{
		"id", "application_id", "key", "name", "description", "status", "start_date", "end_date", "created_at", "updated_at",
	}).AddRow("exp_123", "app_123", "exp-key", "Experiment", nil, "draft", nil, nil, nowTime(), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO experiments (application_id, key, name, description, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, application_id, key, name, description, status, start_date, end_date, created_at, updated_at`)).
		WithArgs("app_123", "exp-key", "Experiment", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(createRows)

	branchRows := pgxmock.NewRows([]string{"id", "experiment_id", "key", "name", "weight", "metadata_json"}).
		AddRow("branch_123", "exp_123", "control", "Control", 0.5, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO branches (experiment_id, key, name, weight, metadata_json)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, experiment_id, key, name, weight, metadata_json`)).
		WithArgs("exp_123", "control", "Control", 0.5, nil).
		WillReturnRows(branchRows)
	mock.ExpectCommit()

	exp, err := store.Create(context.Background(), CreateExperimentParams{
		ApplicationID: "app_123",
		Key:           "exp-key",
		Name:          "Experiment",
		Branches: []CreateBranchParams{
			{Key: "control", Name: "Control", Weight: 0.5},
		},
	})
	if err != nil {
		t.Fatalf("Create with branches returned error: %v", err)
	}
	if len(exp.Branches) != 1 {
		t.Fatalf("expected 1 branch, got %d", len(exp.Branches))
	}
}

func TestExperimentStoreGetUpdateDelete(t *testing.T) {
	mock := newMockPool(t)
	store := NewExperimentStore(mock)

	getRows := pgxmock.NewRows([]string{
		"id", "application_id", "key", "name", "description", "status", "start_date", "end_date", "created_at", "updated_at",
	}).AddRow("exp_123", "app_123", "exp-key", "Experiment", nil, "active", nil, nil, nowTime(), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, application_id, key, name, description, status, start_date, end_date, created_at, updated_at
		FROM experiments
		WHERE id = $1 AND application_id = $2 AND deleted_at IS NULL`)).
		WithArgs("exp_123", "app_123").
		WillReturnRows(getRows)

	updateRows := pgxmock.NewRows([]string{
		"id", "application_id", "key", "name", "description", "status", "start_date", "end_date", "created_at", "updated_at",
	}).AddRow("exp_123", "app_123", "exp-key", "Updated", nil, "paused", nil, nil, nowTime(), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		UPDATE experiments
		SET name = $1, description = $2, status = $3, start_date = $4, end_date = $5, updated_at = NOW()
		WHERE id = $6 AND application_id = $7 AND deleted_at IS NULL
		RETURNING id, application_id, key, name, description, status, start_date, end_date, created_at, updated_at`)).
		WithArgs("Updated", pgxmock.AnyArg(), models.ExperimentStatusPaused, pgxmock.AnyArg(), pgxmock.AnyArg(), "exp_123", "app_123").
		WillReturnRows(updateRows)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE branches
		SET deleted_at = NOW()
		WHERE experiment_id = $1 AND deleted_at IS NULL`)).
		WithArgs("exp_123").
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE experiments
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND application_id = $2 AND deleted_at IS NULL`)).
		WithArgs("exp_123", "app_123").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	if _, err := store.GetByID(context.Background(), "app_123", "exp_123"); err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if _, err := store.Update(context.Background(), "app_123", "exp_123", UpdateExperimentParams{
		Name:   "Updated",
		Status: models.ExperimentStatusPaused,
	}); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if err := store.Delete(context.Background(), "app_123", "exp_123"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestClassifyExperimentErr(t *testing.T) {
	conflictErr := classifyExperimentErr("create", &pgconn.PgError{Code: "23505"})
	if !errors.Is(conflictErr, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", conflictErr)
	}

	notFoundErr := classifyExperimentErr("create", &pgconn.PgError{Code: "23503"})
	if !errors.Is(notFoundErr, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", notFoundErr)
	}
}

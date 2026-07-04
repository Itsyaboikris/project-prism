package store

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestBranchStoreCreateAndGet(t *testing.T) {
	mock := newMockPool(t)
	store := NewBranchStore(mock)

	createRows := pgxmock.NewRows([]string{"id", "experiment_id", "key", "name", "weight", "metadata_json"}).
		AddRow("branch_123", "exp_123", "control", "Control", 0.5, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO branches (experiment_id, key, name, weight, metadata_json)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, experiment_id, key, name, weight, metadata_json`)).
		WithArgs("exp_123", "control", "Control", 0.5, nil).
		WillReturnRows(createRows)

	getRows := pgxmock.NewRows([]string{"id", "experiment_id", "key", "name", "weight", "metadata_json"}).
		AddRow("branch_123", "exp_123", "control", "Control", 0.5, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, experiment_id, key, name, weight, metadata_json
		FROM branches
		WHERE id = $1 AND experiment_id = $2 AND deleted_at IS NULL`)).
		WithArgs("branch_123", "exp_123").
		WillReturnRows(getRows)

	if _, err := store.Create(context.Background(), CreateBranchParams{
		ExperimentID: "exp_123",
		Key:          "control",
		Name:         "Control",
		Weight:       0.5,
	}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := store.GetByID(context.Background(), "exp_123", "branch_123"); err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
}

func TestBranchStoreListByExperimentIDs(t *testing.T) {
	mock := newMockPool(t)
	store := NewBranchStore(mock)

	rows := pgxmock.NewRows([]string{"id", "experiment_id", "key", "name", "weight", "metadata_json"}).
		AddRow("branch_123", "exp_123", "control", "Control", 0.5, nil).
		AddRow("branch_456", "exp_456", "variant", "Variant", 0.5, nil)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, experiment_id, key, name, weight, metadata_json
		FROM branches
		WHERE experiment_id = ANY($1) AND deleted_at IS NULL
		ORDER BY experiment_id, name`)).
		WithArgs([]string{"exp_123", "exp_456"}).
		WillReturnRows(rows)

	got, err := store.ListByExperimentIDs(context.Background(), []string{"exp_123", "exp_456"})
	if err != nil {
		t.Fatalf("ListByExperimentIDs returned error: %v", err)
	}
	if len(got["exp_123"]) != 1 || len(got["exp_456"]) != 1 {
		t.Fatalf("unexpected map: %#v", got)
	}

	empty, err := store.ListByExperimentIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListByExperimentIDs(nil) returned error: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty map, got %#v", empty)
	}
}

func TestBranchStoreUpdateDeleteAndHelpers(t *testing.T) {
	mock := newMockPool(t)
	store := NewBranchStore(mock)

	updateRows := pgxmock.NewRows([]string{"id", "experiment_id", "key", "name", "weight", "metadata_json"}).
		AddRow("branch_123", "exp_123", "control", "Control Updated", 0.7, []byte(`{"ok":true}`))
	mock.ExpectQuery(regexp.QuoteMeta(`
		UPDATE branches
		SET name = $1, weight = $2, metadata_json = $3
		WHERE id = $4 AND experiment_id = $5 AND deleted_at IS NULL
		RETURNING id, experiment_id, key, name, weight, metadata_json`)).
		WithArgs("Control Updated", 0.7, pgxmock.AnyArg(), "branch_123", "exp_123").
		WillReturnRows(updateRows)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE branches
		SET deleted_at = NOW()
		WHERE id = $1 AND experiment_id = $2 AND deleted_at IS NULL`)).
		WithArgs("branch_123", "exp_123").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	metadata := json.RawMessage(`{"ok":true}`)
	if _, err := store.Update(context.Background(), "exp_123", "branch_123", UpdateBranchParams{
		Name:         "Control Updated",
		Weight:       0.7,
		MetadataJSON: metadata,
	}); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if err := store.Delete(context.Background(), "exp_123", "branch_123"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if nilIfEmpty(nil) != nil {
		t.Fatal("expected nilIfEmpty(nil) to return nil")
	}
	if nilIfEmpty(metadata) == nil {
		t.Fatal("expected nilIfEmpty(non-empty) to return value")
	}
}

func TestBranchStoreNotFoundAndClassify(t *testing.T) {
	mock := newMockPool(t)
	store := NewBranchStore(mock)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, experiment_id, key, name, weight, metadata_json
		FROM branches
		WHERE id = $1 AND experiment_id = $2 AND deleted_at IS NULL`)).
		WithArgs("missing", "exp_123").
		WillReturnError(pgx.ErrNoRows)

	_, err := store.GetByID(context.Background(), "exp_123", "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE branches
		SET deleted_at = NOW()
		WHERE id = $1 AND experiment_id = $2 AND deleted_at IS NULL`)).
		WithArgs("missing", "exp_123").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	err = store.Delete(context.Background(), "exp_123", "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	conflictErr := classifyBranchErr("create", &pgconn.PgError{Code: "23505"})
	if !errors.Is(conflictErr, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", conflictErr)
	}

	notFoundErr := classifyBranchErr("create", &pgconn.PgError{Code: "23503"})
	if !errors.Is(notFoundErr, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", notFoundErr)
	}
}

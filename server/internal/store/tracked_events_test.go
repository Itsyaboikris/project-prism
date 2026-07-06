package store

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestTrackedEventStoreCreateAndDelete(t *testing.T) {
	mock := newMockPool(t)
	store := NewTrackedEventStore(mock)

	createRows := pgxmock.NewRows([]string{
		"id", "experiment_id", "key", "name", "description", "created_at", "updated_at",
	}).AddRow("te_123", "exp_123", "button_click", "Button Click", nil, nowTime(), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO tracked_events (experiment_id, key, name, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, experiment_id, key, name, description, created_at, updated_at`)).
		WithArgs("exp_123", "button_click", "Button Click", pgxmock.AnyArg()).
		WillReturnRows(createRows)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE tracked_events
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND experiment_id = $2 AND deleted_at IS NULL`)).
		WithArgs("te_123", "exp_123").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	if _, err := store.Create(context.Background(), CreateTrackedEventParams{
		ExperimentID: "exp_123",
		Key:          "button_click",
		Name:         "Button Click",
	}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := store.Delete(context.Background(), "exp_123", "te_123"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestTrackedEventStoreListByExperimentID(t *testing.T) {
	mock := newMockPool(t)
	store := NewTrackedEventStore(mock)

	rows := pgxmock.NewRows([]string{
		"id", "experiment_id", "key", "name", "description", "created_at", "updated_at", "occurrence_count", "last_occurred_at",
	}).AddRow("te_123", "exp_123", "button_click", "Button Click", nil, nowTime(), nowTime(), int64(2), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			te.id,
			te.experiment_id,
			te.key,
			te.name,
			te.description,
			te.created_at,
			te.updated_at,
			COUNT(e.id)::bigint AS occurrence_count,
			MAX(e.occurred_at) AS last_occurred_at
		FROM tracked_events te
		LEFT JOIN events e
			ON e.experiment_id = te.experiment_id
			AND e.event_name = te.key
		WHERE te.experiment_id = $1
		  AND te.deleted_at IS NULL
		GROUP BY te.id
		ORDER BY te.name`)).
		WithArgs("exp_123").
		WillReturnRows(rows)

	got, err := store.ListByExperimentID(context.Background(), "exp_123")
	if err != nil {
		t.Fatalf("ListByExperimentID returned error: %v", err)
	}
	if len(got) != 1 || got[0].OccurrenceCount != 2 {
		t.Fatalf("unexpected list: %#v", got)
	}
}

func TestTrackedEventStoreGetByIDNotFound(t *testing.T) {
	mock := newMockPool(t)
	store := NewTrackedEventStore(mock)

	mock.ExpectQuery("SELECT").
		WithArgs("te_missing", "exp_123").
		WillReturnError(pgx.ErrNoRows)

	_, err := store.GetByID(context.Background(), "exp_123", "te_missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestTrackedEventStoreIsRegistered(t *testing.T) {
	mock := newMockPool(t)
	store := NewTrackedEventStore(mock)

	rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT EXISTS (
			SELECT 1
			FROM tracked_events
			WHERE experiment_id = $1
			  AND key = $2
			  AND deleted_at IS NULL
		)`)).
		WithArgs("exp_123", "button_click").
		WillReturnRows(rows)

	registered, err := store.IsRegistered(context.Background(), "exp_123", "button_click")
	if err != nil {
		t.Fatalf("IsRegistered returned error: %v", err)
	}
	if !registered {
		t.Fatalf("expected registered=true")
	}
}

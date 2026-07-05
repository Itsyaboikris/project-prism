package store

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestEventStoreCreateWithoutExperiment(t *testing.T) {
	mock := newMockPool(t)
	store := NewEventStore(mock)

	insertRows := pgxmock.NewRows([]string{
		"id", "application_id", "experiment_id", "branch_id", "user_id", "event_name", "properties_json", "occurred_at", "created_at",
	}).AddRow("event_123", "app_123", nil, nil, "user_123", "purchase", nil, nowTime(), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO events (application_id, experiment_id, branch_id, user_id, event_name, properties_json)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, application_id, experiment_id, branch_id, user_id, event_name, properties_json, occurred_at, created_at`)).
		WithArgs("app_123", nil, nil, "user_123", "purchase", nil).
		WillReturnRows(insertRows)

	got, err := store.Create(context.Background(), CreateEventParams{
		ApplicationID: "app_123",
		UserID:        "user_123",
		EventName:     "purchase",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if got.ID != "event_123" || got.EventName != "purchase" {
		t.Fatalf("unexpected event: %#v", got)
	}
}

func TestEventStoreCreateWithExperimentAndAssignment(t *testing.T) {
	mock := newMockPool(t)
	store := NewEventStore(mock)

	experimentRows := pgxmock.NewRows([]string{"id"}).AddRow("exp_123")
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id
		FROM experiments
		WHERE application_id = $1 AND key = $2 AND deleted_at IS NULL`)).
		WithArgs("app_123", "checkout-button-color").
		WillReturnRows(experimentRows)

	assignmentRows := pgxmock.NewRows([]string{"branch_id"}).AddRow("branch_123")
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT a.branch_id
		FROM assignments a
		JOIN branches b ON b.id = a.branch_id
		WHERE a.experiment_id = $1
		  AND a.user_id = $2
		  AND b.deleted_at IS NULL`)).
		WithArgs("exp_123", "user_123").
		WillReturnRows(assignmentRows)

	properties := json.RawMessage(`{"amount":49.99}`)
	insertRows := pgxmock.NewRows([]string{
		"id", "application_id", "experiment_id", "branch_id", "user_id", "event_name", "properties_json", "occurred_at", "created_at",
	}).AddRow("event_123", "app_123", "exp_123", "branch_123", "user_123", "purchase", []byte(properties), nowTime(), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO events (application_id, experiment_id, branch_id, user_id, event_name, properties_json)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, application_id, experiment_id, branch_id, user_id, event_name, properties_json, occurred_at, created_at`)).
		WithArgs("app_123", "exp_123", "branch_123", "user_123", "purchase", []byte(properties)).
		WillReturnRows(insertRows)

	got, err := store.Create(context.Background(), CreateEventParams{
		ApplicationID:  "app_123",
		UserID:         "user_123",
		EventName:      "purchase",
		ExperimentKey:  "checkout-button-color",
		PropertiesJSON: properties,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if got.ExperimentID == nil || *got.ExperimentID != "exp_123" {
		t.Fatalf("expected experiment id exp_123, got %#v", got.ExperimentID)
	}
	if got.BranchID == nil || *got.BranchID != "branch_123" {
		t.Fatalf("expected branch id branch_123, got %#v", got.BranchID)
	}
}

func TestEventStoreCreateExperimentNotFound(t *testing.T) {
	mock := newMockPool(t)
	store := NewEventStore(mock)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id
		FROM experiments
		WHERE application_id = $1 AND key = $2 AND deleted_at IS NULL`)).
		WithArgs("app_123", "missing").
		WillReturnError(pgx.ErrNoRows)

	_, err := store.Create(context.Background(), CreateEventParams{
		ApplicationID: "app_123",
		UserID:        "user_123",
		EventName:     "purchase",
		ExperimentKey: "missing",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestEventStoreListByExperiment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := newMockPool(t)
		store := NewEventStore(mock)

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

		eventRows := pgxmock.NewRows([]string{
			"id", "application_id", "experiment_id", "branch_id", "user_id", "event_name",
			"properties_json", "occurred_at", "created_at", "key", "name",
		}).AddRow("event_123", "app_123", "exp_123", "branch_123", "user_123", "purchase", nil, nowTime(), nowTime(), "control", "Control")
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT e.id, e.application_id, e.experiment_id, e.branch_id, e.user_id, e.event_name,
		       e.properties_json, e.occurred_at, e.created_at, b.key, b.name
		FROM events e
		LEFT JOIN branches b ON b.id = e.branch_id
		WHERE e.application_id = $1
		  AND e.experiment_id = $2
		  AND ($3 = '' OR e.event_name = $3)
		ORDER BY e.occurred_at DESC, e.id DESC
		LIMIT $4 OFFSET $5`)).
			WithArgs("app_123", "exp_123", "purchase", 100, 0).
			WillReturnRows(eventRows)

		got, err := store.ListByExperiment(context.Background(), ListEventsParams{
			ApplicationID: "app_123",
			ExperimentID:  "exp_123",
			EventName:     "purchase",
			Limit:         100,
			Offset:        0,
		})
		if err != nil {
			t.Fatalf("ListByExperiment returned error: %v", err)
		}
		if got.ExperimentID != "exp_123" || len(got.Events) != 1 {
			t.Fatalf("unexpected events view: %#v", got)
		}
		if got.Events[0].BranchKey == nil || *got.Events[0].BranchKey != "control" {
			t.Fatalf("expected branch key control, got %#v", got.Events[0].BranchKey)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := newMockPool(t)
		store := NewEventStore(mock)

		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, key, name, status
		FROM experiments
		WHERE application_id = $1
		  AND id = $2
		  AND deleted_at IS NULL`)).
			WithArgs("app_123", "exp_123").
			WillReturnError(pgx.ErrNoRows)

		_, err := store.ListByExperiment(context.Background(), ListEventsParams{
			ApplicationID: "app_123",
			ExperimentID:  "exp_123",
			Limit:         100,
		})
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestEventStoreGetEventMetricsByExperiment(t *testing.T) {
	mock := newMockPool(t)
	store := NewEventStore(mock)

	metricsRows := pgxmock.NewRows([]string{"id", "count", "unique_users"}).
		AddRow("branch_123", int64(3), int64(2)).
		AddRow("branch_456", int64(0), int64(0))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT b.id, COUNT(e.id)::bigint, COUNT(DISTINCT e.user_id)::bigint
		FROM branches b
		LEFT JOIN events e
		  ON e.branch_id = b.id
		 AND e.experiment_id = $1
		 AND e.event_name = $2
		WHERE b.experiment_id = $1
		  AND b.deleted_at IS NULL
		GROUP BY b.id`)).
		WithArgs("exp_123", "purchase").
		WillReturnRows(metricsRows)

	got, err := store.GetEventMetricsByExperiment(context.Background(), "exp_123", "purchase")
	if err != nil {
		t.Fatalf("GetEventMetricsByExperiment returned error: %v", err)
	}
	if got["branch_123"].EventCount != 3 || got["branch_123"].UniqueEventUsers != 2 {
		t.Fatalf("unexpected control metrics: %#v", got["branch_123"])
	}
	if got["branch_456"].EventCount != 0 || got["branch_456"].UniqueEventUsers != 0 {
		t.Fatalf("expected zero metrics for branch without events, got %#v", got["branch_456"])
	}
}

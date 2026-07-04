package store

import (
	"context"
	"errors"
	"regexp"
	"testing"

	pgxmock "github.com/pashagolub/pgxmock/v4"
	"project-prism/server/internal/models"
)

func newMockPool(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("new pgxmock pool: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
		mock.Close()
	})
	return mock
}

func TestApplicationStoreCreate(t *testing.T) {
	mock := newMockPool(t)
	store := NewApplicationStore(mock)

	rows := pgxmock.NewRows([]string{"id", "name", "api_key", "status", "created_at", "updated_at"}).
		AddRow("app_123", "My App", "prism_123", "active", nowTime(), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO applications (name, api_key)
		VALUES ($1, $2)
		RETURNING id, name, api_key, status, created_at, updated_at`)).
		WithArgs("My App", "prism_123").
		WillReturnRows(rows)

	app, err := store.Create(context.Background(), "My App", "prism_123")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if app.ID != "app_123" || app.Status != models.ApplicationStatusActive {
		t.Fatalf("unexpected app: %#v", app)
	}
}

func TestApplicationStoreList(t *testing.T) {
	mock := newMockPool(t)
	store := NewApplicationStore(mock)

	rows := pgxmock.NewRows([]string{"id", "name", "api_key", "status", "created_at", "updated_at"}).
		AddRow("app_123", "My App", "prism_123", "active", nowTime(), nowTime()).
		AddRow("app_456", "Other App", "prism_456", "inactive", nowTime(), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, api_key, status, created_at, updated_at
		FROM applications
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC`)).
		WillReturnRows(rows)

	apps, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(apps))
	}
}

func TestApplicationStoreGettersAndUpdate(t *testing.T) {
	mock := newMockPool(t)
	store := NewApplicationStore(mock)

	byIDRows := pgxmock.NewRows([]string{"id", "name", "api_key", "status", "created_at", "updated_at"}).
		AddRow("app_123", "My App", "prism_123", "active", nowTime(), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, api_key, status, created_at, updated_at
		FROM applications
		WHERE id = $1 AND deleted_at IS NULL`)).
		WithArgs("app_123").
		WillReturnRows(byIDRows)

	byKeyRows := pgxmock.NewRows([]string{"id", "name", "api_key", "status", "created_at", "updated_at"}).
		AddRow("app_123", "My App", "prism_123", "active", nowTime(), nowTime())
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, name, api_key, status, created_at, updated_at
		FROM applications
		WHERE api_key = $1 AND deleted_at IS NULL`)).
		WithArgs("prism_123").
		WillReturnRows(byKeyRows)

	updateRows := pgxmock.NewRows([]string{"id", "name", "api_key", "status", "created_at", "updated_at"}).
		AddRow("app_123", "Renamed", "prism_123", "inactive", nowTime(), nowTime())
	status := models.ApplicationStatusInactive
	mock.ExpectQuery(regexp.QuoteMeta(`
		UPDATE applications
		SET name = $1, status = COALESCE($2, status), updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
		RETURNING id, name, api_key, status, created_at, updated_at`)).
		WithArgs("Renamed", &status, "app_123").
		WillReturnRows(updateRows)

	if _, err := store.GetByID(context.Background(), "app_123"); err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if _, err := store.GetByAPIKey(context.Background(), "prism_123"); err != nil {
		t.Fatalf("GetByAPIKey returned error: %v", err)
	}
	if _, err := store.Update(context.Background(), "app_123", UpdateApplicationParams{Name: "Renamed", Status: &status}); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
}

func TestApplicationStoreDelete(t *testing.T) {
	mock := newMockPool(t)
	store := NewApplicationStore(mock)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE branches b
		SET deleted_at = NOW()
		FROM experiments e
		WHERE b.experiment_id = e.id
		  AND e.application_id = $1
		  AND b.deleted_at IS NULL
		  AND e.deleted_at IS NULL`)).
		WithArgs("app_123").
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE experiments
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE application_id = $1 AND deleted_at IS NULL`)).
		WithArgs("app_123").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE applications
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`)).
		WithArgs("app_123").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	if err := store.Delete(context.Background(), "app_123"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestApplicationStoreDeleteReturnsNotFound(t *testing.T) {
	mock := newMockPool(t)
	store := NewApplicationStore(mock)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE branches b
		SET deleted_at = NOW()
		FROM experiments e
		WHERE b.experiment_id = e.id
		  AND e.application_id = $1
		  AND b.deleted_at IS NULL
		  AND e.deleted_at IS NULL`)).
		WithArgs("app_123").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE experiments
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE application_id = $1 AND deleted_at IS NULL`)).
		WithArgs("app_123").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE applications
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`)).
		WithArgs("app_123").
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	mock.ExpectRollback()

	err := store.Delete(context.Background(), "app_123")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

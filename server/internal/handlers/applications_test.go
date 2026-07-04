package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"project-prism/server/internal/models"
	"project-prism/server/internal/store"
)

type fakeApplicationStore struct {
	createFn func(context.Context, string, string) (*models.Application, error)
	listFn   func(context.Context) ([]*models.Application, error)
	getFn    func(context.Context, string) (*models.Application, error)
	updateFn func(context.Context, string, store.UpdateApplicationParams) (*models.Application, error)
	deleteFn func(context.Context, string) error
}

func (f *fakeApplicationStore) Create(ctx context.Context, name, apiKey string) (*models.Application, error) {
	return f.createFn(ctx, name, apiKey)
}

func (f *fakeApplicationStore) List(ctx context.Context) ([]*models.Application, error) {
	return f.listFn(ctx)
}

func (f *fakeApplicationStore) GetByID(ctx context.Context, id string) (*models.Application, error) {
	return f.getFn(ctx, id)
}

func (f *fakeApplicationStore) Update(ctx context.Context, id string, p store.UpdateApplicationParams) (*models.Application, error) {
	return f.updateFn(ctx, id, p)
}

func (f *fakeApplicationStore) Delete(ctx context.Context, id string) error {
	return f.deleteFn(ctx, id)
}

func TestApplicationHandlerCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var gotName, gotAPIKey string
		handler := NewApplicationHandler(&fakeApplicationStore{
			createFn: func(_ context.Context, name, apiKey string) (*models.Application, error) {
				gotName, gotAPIKey = name, apiKey
				return &models.Application{ID: "app_123", Name: name, APIKey: apiKey, Status: models.ApplicationStatusActive}, nil
			},
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/applications", strings.NewReader(`{"name":"  My App  "}`))
		rec := httptest.NewRecorder()

		handler.Create(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
		}
		if gotName != "My App" {
			t.Fatalf("expected trimmed name %q, got %q", "My App", gotName)
		}
		if gotAPIKey == "" {
			t.Fatal("expected generated api key")
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{})
		rec := httptest.NewRecorder()
		handler.Create(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{")))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{})
		rec := httptest.NewRecorder()
		handler.Create(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"   "}`)))
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{
			createFn: func(context.Context, string, string) (*models.Application, error) {
				return nil, errors.New("boom")
			},
		})
		rec := httptest.NewRecorder()
		handler.Create(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"My App"}`)))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})
}

func TestApplicationHandlerList(t *testing.T) {
	t.Run("success with nil slice", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{
			listFn: func(context.Context) ([]*models.Application, error) { return nil, nil },
		})
		rec := httptest.NewRecorder()

		handler.List(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var apps []models.Application
		if err := json.Unmarshal(rec.Body.Bytes(), &apps); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if len(apps) != 0 {
			t.Fatalf("expected empty array, got %#v", apps)
		}
	})

	t.Run("store error", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{
			listFn: func(context.Context) ([]*models.Application, error) { return nil, errors.New("boom") },
		})
		rec := httptest.NewRecorder()
		handler.List(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})
}

func TestApplicationHandlerGetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{
			getFn: func(_ context.Context, id string) (*models.Application, error) {
				return &models.Application{ID: id, Name: "My App", Status: models.ApplicationStatusActive}, nil
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/api/v1/applications/app_123", "", map[string]string{"id": "app_123"})

		handler.GetByID(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{
			getFn: func(context.Context, string) (*models.Application, error) { return nil, store.ErrNotFound },
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/", "", map[string]string{"id": "missing"})
		handler.GetByID(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("unexpected error", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{
			getFn: func(context.Context, string) (*models.Application, error) { return nil, errors.New("boom") },
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/", "", map[string]string{"id": "app_123"})
		handler.GetByID(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})
}

func TestApplicationHandlerUpdate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var gotParams store.UpdateApplicationParams
		handler := NewApplicationHandler(&fakeApplicationStore{
			updateFn: func(_ context.Context, id string, p store.UpdateApplicationParams) (*models.Application, error) {
				gotParams = p
				return &models.Application{ID: id, Name: p.Name, Status: *p.Status}, nil
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodPut, "/", `{"name":"  Renamed  ","status":"inactive"}`, map[string]string{"id": "app_123"})

		handler.Update(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if gotParams.Name != "Renamed" {
			t.Fatalf("expected trimmed name %q, got %q", "Renamed", gotParams.Name)
		}
		if gotParams.Status == nil || *gotParams.Status != models.ApplicationStatusInactive {
			t.Fatalf("expected inactive status, got %#v", gotParams.Status)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodPut, "/", "{", map[string]string{"id": "app_123"})
		handler.Update(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodPut, "/", `{"name":" "}`, map[string]string{"id": "app_123"})
		handler.Update(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodPut, "/", `{"name":"App","status":"archived"}`, map[string]string{"id": "app_123"})
		handler.Update(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{
			updateFn: func(context.Context, string, store.UpdateApplicationParams) (*models.Application, error) {
				return nil, store.ErrNotFound
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodPut, "/", `{"name":"App"}`, map[string]string{"id": "app_123"})
		handler.Update(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestApplicationHandlerDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var gotID string
		handler := NewApplicationHandler(&fakeApplicationStore{
			deleteFn: func(_ context.Context, id string) error {
				gotID = id
				return nil
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodDelete, "/", "", map[string]string{"id": "app_123"})

		handler.Delete(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
		}
		if gotID != "app_123" {
			t.Fatalf("expected delete id %q, got %q", "app_123", gotID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{
			deleteFn: func(context.Context, string) error { return store.ErrNotFound },
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodDelete, "/", "", map[string]string{"id": "app_123"})
		handler.Delete(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("unexpected error", func(t *testing.T) {
		handler := NewApplicationHandler(&fakeApplicationStore{
			deleteFn: func(context.Context, string) error { return errors.New("boom") },
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodDelete, "/", "", map[string]string{"id": "app_123"})
		handler.Delete(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})
}

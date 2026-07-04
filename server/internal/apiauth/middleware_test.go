package apiauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-prism/server/internal/models"
	"project-prism/server/internal/store"
)

type fakeApplicationLookup struct {
	app         *models.Application
	err         error
	receivedKey string
}

func (f *fakeApplicationLookup) GetByAPIKey(_ context.Context, apiKey string) (*models.Application, error) {
	f.receivedKey = apiKey
	return f.app, f.err
}

func TestRequireAPIKeyRejectsMissingKey(t *testing.T) {
	mw := NewMiddleware(&fakeApplicationLookup{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assign", nil)
	rec := httptest.NewRecorder()

	mw.RequireAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "missing api key" {
		t.Fatalf("expected missing api key error, got %q", body["error"])
	}
}

func TestRequireAPIKeyRejectsInvalidKey(t *testing.T) {
	lookup := &fakeApplicationLookup{err: store.ErrNotFound}
	mw := NewMiddleware(lookup)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assign", nil)
	req.Header.Set("X-API-Key", "bad_key")
	rec := httptest.NewRecorder()

	mw.RequireAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if lookup.receivedKey != "bad_key" {
		t.Fatalf("expected lookup key %q, got %q", "bad_key", lookup.receivedKey)
	}
}

func TestRequireAPIKeyRejectsInactiveApplication(t *testing.T) {
	lookup := &fakeApplicationLookup{
		app: &models.Application{
			ID:     "app_123",
			Status: models.ApplicationStatusInactive,
		},
	}
	mw := NewMiddleware(lookup)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assign", nil)
	req.Header.Set("X-API-Key", "prism_test")
	rec := httptest.NewRecorder()

	mw.RequireAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestRequireAPIKeyUsesBearerTokenAndInjectsApplication(t *testing.T) {
	wantApp := &models.Application{
		ID:     "app_123",
		Status: models.ApplicationStatusActive,
	}
	lookup := &fakeApplicationLookup{app: wantApp}
	mw := NewMiddleware(lookup)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assign", nil)
	req.Header.Set("Authorization", "Bearer prism_bearer_token")
	rec := httptest.NewRecorder()

	var gotApp *models.Application
	mw.RequireAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		gotApp, ok = ApplicationFromContext(r.Context())
		if !ok {
			t.Fatal("expected application in context")
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if lookup.receivedKey != "prism_bearer_token" {
		t.Fatalf("expected bearer token to be used, got %q", lookup.receivedKey)
	}
	if gotApp == nil || gotApp.ID != wantApp.ID {
		t.Fatalf("expected application %q in context, got %#v", wantApp.ID, gotApp)
	}
}

func TestRequireAPIKeyHandlesLookupFailure(t *testing.T) {
	lookup := &fakeApplicationLookup{err: errors.New("db unavailable")}
	mw := NewMiddleware(lookup)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assign", nil)
	req.Header.Set("X-API-Key", "prism_test")
	rec := httptest.NewRecorder()

	mw.RequireAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

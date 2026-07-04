package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"project-prism/server/internal/apiauth"
	"project-prism/server/internal/models"
	"project-prism/server/internal/store"
)

type fakeAssignmentStore struct {
	branch     *models.Branch
	err        error
	lastParams store.AssignParams
}

func (f *fakeAssignmentStore) Assign(_ context.Context, p store.AssignParams) (*models.Branch, error) {
	f.lastParams = p
	return f.branch, f.err
}

func TestAssignmentHandlerCreateSuccess(t *testing.T) {
	store := &fakeAssignmentStore{
		branch: &models.Branch{
			ID:           "branch_123",
			ExperimentID: "exp_123",
			Key:          "variant-a",
			Name:         "Variant A",
			Weight:       0.7,
		},
	}
	handler := NewAssignmentHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assign", strings.NewReader(`{
		"user_id": "  user_123  ",
		"experiment_key": "  checkout-button-color  "
	}`))
	req = req.WithContext(apiauth.WithApplication(req.Context(), &models.Application{ID: "app_123"}))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if store.lastParams.ApplicationID != "app_123" {
		t.Fatalf("expected application id %q, got %q", "app_123", store.lastParams.ApplicationID)
	}
	if store.lastParams.UserID != "user_123" {
		t.Fatalf("expected trimmed user id %q, got %q", "user_123", store.lastParams.UserID)
	}
	if store.lastParams.ExperimentKey != "checkout-button-color" {
		t.Fatalf("expected trimmed experiment key %q, got %q", "checkout-button-color", store.lastParams.ExperimentKey)
	}

	var body models.Branch
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.ID != "branch_123" {
		t.Fatalf("expected branch id %q, got %q", "branch_123", body.ID)
	}
}

func TestAssignmentHandlerCreateRequiresApplicationContext(t *testing.T) {
	handler := NewAssignmentHandler(&fakeAssignmentStore{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assign", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestAssignmentHandlerCreateValidatesBody(t *testing.T) {
	testCases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "missing user id", body: `{"experiment_key":"exp-key"}`, wantStatus: http.StatusUnprocessableEntity},
		{name: "missing experiment key", body: `{"user_id":"user_123"}`, wantStatus: http.StatusUnprocessableEntity},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewAssignmentHandler(&fakeAssignmentStore{})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/assign", strings.NewReader(tc.body))
			req = req.WithContext(apiauth.WithApplication(req.Context(), &models.Application{ID: "app_123"}))
			rec := httptest.NewRecorder()

			handler.Create(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestAssignmentHandlerCreateMapsStoreErrors(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "not found", err: store.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "not eligible", err: store.ErrNotEligible, wantStatus: http.StatusConflict},
		{name: "misconfigured", err: store.ErrMisconfigured, wantStatus: http.StatusConflict},
		{name: "unexpected", err: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewAssignmentHandler(&fakeAssignmentStore{err: tc.err})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/assign", strings.NewReader(`{
				"user_id":"user_123",
				"experiment_key":"exp-key"
			}`))
			req = req.WithContext(apiauth.WithApplication(req.Context(), &models.Application{ID: "app_123"}))
			rec := httptest.NewRecorder()

			handler.Create(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

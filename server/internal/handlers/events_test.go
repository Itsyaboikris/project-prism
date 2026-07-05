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

type fakeEventStore struct {
	event      *models.Event
	eventsView *models.ExperimentEventsView
	createErr  error
	listErr    error
	lastCreate store.CreateEventParams
	lastList   store.ListEventsParams
}

func (f *fakeEventStore) Create(_ context.Context, p store.CreateEventParams) (*models.Event, error) {
	f.lastCreate = p
	return f.event, f.createErr
}

func (f *fakeEventStore) ListByExperiment(_ context.Context, p store.ListEventsParams) (*models.ExperimentEventsView, error) {
	f.lastList = p
	return f.eventsView, f.listErr
}

func TestEventHandlerCreateSuccess(t *testing.T) {
	experimentID := "exp_123"
	branchID := "branch_123"
	store := &fakeEventStore{
		event: &models.Event{
			ID:           "event_123",
			UserID:       "user_123",
			EventName:    "purchase",
			ExperimentID: &experimentID,
			BranchID:     &branchID,
			Properties:   json.RawMessage(`{"amount":49.99}`),
		},
	}
	handler := NewEventHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(`{
		"user_id": "  user_123  ",
		"event_name": "  purchase  ",
		"experiment_key": "  checkout-button-color  ",
		"properties": { "amount": 49.99 }
	}`))
	req = req.WithContext(apiauth.WithApplication(req.Context(), &models.Application{ID: "app_123"}))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if store.lastCreate.ApplicationID != "app_123" {
		t.Fatalf("expected application id %q, got %q", "app_123", store.lastCreate.ApplicationID)
	}
	if store.lastCreate.UserID != "user_123" || store.lastCreate.EventName != "purchase" {
		t.Fatalf("unexpected create params: %#v", store.lastCreate)
	}
	if store.lastCreate.ExperimentKey != "checkout-button-color" {
		t.Fatalf("expected trimmed experiment key, got %q", store.lastCreate.ExperimentKey)
	}

	var body models.Event
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.ID != "event_123" || body.EventName != "purchase" {
		t.Fatalf("unexpected response: %#v", body)
	}
}

func TestEventHandlerCreateRequiresApplicationContext(t *testing.T) {
	handler := NewEventHandler(&fakeEventStore{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestEventHandlerCreateValidatesBody(t *testing.T) {
	testCases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "missing user id", body: `{"event_name":"purchase"}`, wantStatus: http.StatusUnprocessableEntity},
		{name: "missing event name", body: `{"user_id":"user_123"}`, wantStatus: http.StatusUnprocessableEntity},
		{name: "invalid properties", body: `{"user_id":"user_123","event_name":"purchase","properties":[]}`, wantStatus: http.StatusUnprocessableEntity},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewEventHandler(&fakeEventStore{})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(tc.body))
			req = req.WithContext(apiauth.WithApplication(req.Context(), &models.Application{ID: "app_123"}))
			rec := httptest.NewRecorder()

			handler.Create(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestEventHandlerCreateMapsStoreErrors(t *testing.T) {
	testCases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "not found", err: store.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "unexpected", err: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewEventHandler(&fakeEventStore{createErr: tc.err})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(`{
				"user_id":"user_123",
				"event_name":"purchase",
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

func TestEventHandlerListByExperiment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := NewEventHandler(&fakeEventStore{
			eventsView: &models.ExperimentEventsView{
				ExperimentID:     "exp_123",
				ExperimentKey:    "checkout-button-color",
				ExperimentName:   "Checkout Button Color",
				ExperimentStatus: models.ExperimentStatusActive,
				Events: []*models.ExperimentEventListItem{
					{ID: "event_123", UserID: "user_123", EventName: "purchase"},
				},
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/?event_name=purchase&limit=50&offset=10", "", map[string]string{"appID": "app_123", "id": "exp_123"})

		handler.ListByExperiment(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body models.ExperimentEventsView
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if body.ExperimentID != "exp_123" || len(body.Events) != 1 {
			t.Fatalf("unexpected response: %#v", body)
		}
	})

	t.Run("invalid limit", func(t *testing.T) {
		handler := NewEventHandler(&fakeEventStore{})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/?limit=abc", "", map[string]string{"appID": "app_123", "id": "exp_123"})

		handler.ListByExperiment(rec, req)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := NewEventHandler(&fakeEventStore{listErr: store.ErrNotFound})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/", "", map[string]string{"appID": "app_123", "id": "exp_123"})

		handler.ListByExperiment(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

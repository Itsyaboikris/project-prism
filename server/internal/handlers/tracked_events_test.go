package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"project-prism/server/internal/models"
	"project-prism/server/internal/store"
)

type fakeTrackedEventStore struct {
	listFn   func(context.Context, string) ([]*models.TrackedEvent, error)
	createFn func(context.Context, store.CreateTrackedEventParams) (*models.TrackedEvent, error)
	updateFn func(context.Context, string, string, store.UpdateTrackedEventParams) (*models.TrackedEvent, error)
	deleteFn func(context.Context, string, string) error
}

func (f *fakeTrackedEventStore) ListByExperimentID(ctx context.Context, experimentID string) ([]*models.TrackedEvent, error) {
	return f.listFn(ctx, experimentID)
}
func (f *fakeTrackedEventStore) GetByID(ctx context.Context, experimentID, id string) (*models.TrackedEvent, error) {
	return nil, nil
}
func (f *fakeTrackedEventStore) Create(ctx context.Context, p store.CreateTrackedEventParams) (*models.TrackedEvent, error) {
	return f.createFn(ctx, p)
}
func (f *fakeTrackedEventStore) Update(ctx context.Context, experimentID, id string, p store.UpdateTrackedEventParams) (*models.TrackedEvent, error) {
	return f.updateFn(ctx, experimentID, id, p)
}
func (f *fakeTrackedEventStore) Delete(ctx context.Context, experimentID, id string) error {
	return f.deleteFn(ctx, experimentID, id)
}

type fakeTrackedEventExperimentStore struct {
	getFn func(context.Context, string, string) (*models.Experiment, error)
}

func (f *fakeTrackedEventExperimentStore) GetByID(ctx context.Context, applicationID, id string) (*models.Experiment, error) {
	return f.getFn(ctx, applicationID, id)
}

func TestTrackedEventHandlerCreate(t *testing.T) {
	longKey := strings.Repeat("k", trackedEventKeyMaxLength+1)
	longName := strings.Repeat("n", trackedEventNameMaxLength+1)
	longDescription := strings.Repeat("d", trackedEventDescriptionMaxLength+1)

	t.Run("success", func(t *testing.T) {
		var got store.CreateTrackedEventParams
		handler := NewTrackedEventHandler(&fakeTrackedEventStore{
			createFn: func(_ context.Context, p store.CreateTrackedEventParams) (*models.TrackedEvent, error) {
				got = p
				return &models.TrackedEvent{ID: "te_123", Key: p.Key, Name: p.Name}, nil
			},
		}, &fakeTrackedEventExperimentStore{
			getFn: func(context.Context, string, string) (*models.Experiment, error) {
				return &models.Experiment{ID: "exp_123"}, nil
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodPost, "/", `{"key":" button_click ","name":" Button Click ","description":" CTA "}`, map[string]string{
			"appID": "app_123",
			"id":    "exp_123",
		})

		handler.Create(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
		}
		if got.ExperimentID != "exp_123" || got.Key != "button_click" || got.Name != "Button Click" {
			t.Fatalf("unexpected params: %#v", got)
		}
		if got.Description == nil || *got.Description != "CTA" {
			t.Fatalf("unexpected description: %#v", got.Description)
		}
	})

	testCases := []struct {
		name       string
		body       string
		verifyErr  error
		createErr  error
		wantStatus int
	}{
		{"experiment missing", `{"key":"button_click","name":"Button Click"}`, store.ErrNotFound, nil, http.StatusNotFound},
		{"verify error", `{"key":"button_click","name":"Button Click"}`, errors.New("boom"), nil, http.StatusInternalServerError},
		{"invalid body", "{", nil, nil, http.StatusBadRequest},
		{"missing key", `{"name":"Button Click"}`, nil, nil, http.StatusUnprocessableEntity},
		{"key too long", `{"key":"` + longKey + `","name":"Button Click"}`, nil, nil, http.StatusUnprocessableEntity},
		{"missing name", `{"key":"button_click"}`, nil, nil, http.StatusUnprocessableEntity},
		{"name too long", `{"key":"button_click","name":"` + longName + `"}`, nil, nil, http.StatusUnprocessableEntity},
		{"description too long", `{"key":"button_click","name":"Button Click","description":"` + longDescription + `"}`, nil, nil, http.StatusUnprocessableEntity},
		{"conflict", `{"key":"button_click","name":"Button Click"}`, nil, store.ErrConflict, http.StatusConflict},
		{"create error", `{"key":"button_click","name":"Button Click"}`, nil, errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewTrackedEventHandler(&fakeTrackedEventStore{
				createFn: func(context.Context, store.CreateTrackedEventParams) (*models.TrackedEvent, error) {
					return nil, tc.createErr
				},
			}, &fakeTrackedEventExperimentStore{
				getFn: func(context.Context, string, string) (*models.Experiment, error) {
					return &models.Experiment{ID: "exp_123"}, tc.verifyErr
				},
			})
			rec := httptest.NewRecorder()
			req := newRequestWithURLParams(http.MethodPost, "/", tc.body, map[string]string{
				"appID": "app_123",
				"id":    "exp_123",
			})

			handler.Create(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d body=%s", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestTrackedEventHandlerList(t *testing.T) {
	handler := NewTrackedEventHandler(&fakeTrackedEventStore{
		listFn: func(_ context.Context, experimentID string) ([]*models.TrackedEvent, error) {
			if experimentID != "exp_123" {
				t.Fatalf("unexpected experiment id: %s", experimentID)
			}
			return []*models.TrackedEvent{{ID: "te_123", Key: "button_click", OccurrenceCount: 3}}, nil
		},
	}, &fakeTrackedEventExperimentStore{
		getFn: func(context.Context, string, string) (*models.Experiment, error) {
			return &models.Experiment{ID: "exp_123"}, nil
		},
	})

	rec := httptest.NewRecorder()
	req := newRequestWithURLParams(http.MethodGet, "/", "", map[string]string{
		"appID": "app_123",
		"id":    "exp_123",
	})

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestTrackedEventHandlerDelete(t *testing.T) {
	handler := NewTrackedEventHandler(&fakeTrackedEventStore{
		deleteFn: func(_ context.Context, experimentID, id string) error {
			if experimentID != "exp_123" || id != "te_123" {
				t.Fatalf("unexpected ids: %s %s", experimentID, id)
			}
			return nil
		},
	}, &fakeTrackedEventExperimentStore{
		getFn: func(context.Context, string, string) (*models.Experiment, error) {
			return &models.Experiment{ID: "exp_123"}, nil
		},
	})

	rec := httptest.NewRecorder()
	req := newRequestWithURLParams(http.MethodDelete, "/", "", map[string]string{
		"appID":          "app_123",
		"id":             "exp_123",
		"trackedEventID": "te_123",
	})

	handler.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

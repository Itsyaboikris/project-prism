package handlers

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

type fakeExperimentStore struct {
	createFn func(context.Context, store.CreateExperimentParams) (*models.Experiment, error)
	listFn   func(context.Context, string) ([]*models.Experiment, error)
	getFn    func(context.Context, string, string) (*models.Experiment, error)
	updateFn func(context.Context, string, string, store.UpdateExperimentParams) (*models.Experiment, error)
	deleteFn func(context.Context, string, string) error
}

func (f *fakeExperimentStore) Create(ctx context.Context, p store.CreateExperimentParams) (*models.Experiment, error) {
	return f.createFn(ctx, p)
}
func (f *fakeExperimentStore) List(ctx context.Context, appID string) ([]*models.Experiment, error) {
	return f.listFn(ctx, appID)
}
func (f *fakeExperimentStore) GetByID(ctx context.Context, appID, id string) (*models.Experiment, error) {
	return f.getFn(ctx, appID, id)
}
func (f *fakeExperimentStore) Update(ctx context.Context, appID, id string, p store.UpdateExperimentParams) (*models.Experiment, error) {
	return f.updateFn(ctx, appID, id, p)
}
func (f *fakeExperimentStore) Delete(ctx context.Context, appID, id string) error {
	return f.deleteFn(ctx, appID, id)
}

type fakeExperimentBranchStore struct {
	listFn func(context.Context, []string) (map[string][]*models.Branch, error)
}

func (f *fakeExperimentBranchStore) ListByExperimentIDs(ctx context.Context, ids []string) (map[string][]*models.Branch, error) {
	return f.listFn(ctx, ids)
}

func TestExperimentHandlerCreate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var got store.CreateExperimentParams
		handler := NewExperimentHandler(&fakeExperimentStore{
			createFn: func(_ context.Context, p store.CreateExperimentParams) (*models.Experiment, error) {
				got = p
				return &models.Experiment{ID: "exp_123", ApplicationID: p.ApplicationID, Key: p.Key, Name: p.Name}, nil
			},
		}, &fakeExperimentBranchStore{})
		req := newRequestWithURLParams(http.MethodPost, "/", `{
			"key":"  exp-key  ",
			"name":"  Experiment  ",
			"branches":[{"key":" control ","name":" Control ","weight":0.4}]
		}`, map[string]string{"appID": "app_123"})
		rec := httptest.NewRecorder()

		handler.Create(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
		}
		if got.ApplicationID != "app_123" || got.Key != "exp-key" || got.Name != "Experiment" {
			t.Fatalf("unexpected params: %#v", got)
		}
		if len(got.Branches) != 1 || got.Branches[0].Key != "control" || got.Branches[0].Name != "Control" {
			t.Fatalf("unexpected branch params: %#v", got.Branches)
		}
	})

	testCases := []struct {
		name       string
		body       string
		err        error
		wantStatus int
	}{
		{"invalid body", "{", nil, http.StatusBadRequest},
		{"missing key", `{"name":"Experiment"}`, nil, http.StatusUnprocessableEntity},
		{"missing name", `{"key":"exp-key"}`, nil, http.StatusUnprocessableEntity},
		{"missing branch key", `{"key":"exp-key","name":"Experiment","branches":[{"name":"Control","weight":0.5}]}`, nil, http.StatusUnprocessableEntity},
		{"missing branch name", `{"key":"exp-key","name":"Experiment","branches":[{"key":"control","weight":0.5}]}`, nil, http.StatusUnprocessableEntity},
		{"bad branch weight", `{"key":"exp-key","name":"Experiment","branches":[{"key":"control","name":"Control","weight":2}]}`, nil, http.StatusUnprocessableEntity},
		{"app not found", `{"key":"exp-key","name":"Experiment"}`, store.ErrNotFound, http.StatusNotFound},
		{"app inactive", `{"key":"exp-key","name":"Experiment"}`, store.ErrInactive, http.StatusConflict},
		{"conflict", `{"key":"exp-key","name":"Experiment"}`, store.ErrConflict, http.StatusConflict},
		{"unexpected", `{"key":"exp-key","name":"Experiment"}`, errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewExperimentHandler(&fakeExperimentStore{
				createFn: func(context.Context, store.CreateExperimentParams) (*models.Experiment, error) {
					return nil, tc.err
				},
			}, &fakeExperimentBranchStore{})
			rec := httptest.NewRecorder()
			req := newRequestWithURLParams(http.MethodPost, "/", tc.body, map[string]string{"appID": "app_123"})
			handler.Create(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestExperimentHandlerList(t *testing.T) {
	t.Run("success empty list", func(t *testing.T) {
		handler := NewExperimentHandler(&fakeExperimentStore{
			listFn: func(context.Context, string) ([]*models.Experiment, error) { return []*models.Experiment{}, nil },
		}, &fakeExperimentBranchStore{})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/", "", map[string]string{"appID": "app_123"})

		handler.List(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var body []models.Experiment
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if len(body) != 0 {
			t.Fatalf("expected empty list, got %#v", body)
		}
	})

	t.Run("success with branches", func(t *testing.T) {
		handler := NewExperimentHandler(&fakeExperimentStore{
			listFn: func(context.Context, string) ([]*models.Experiment, error) {
					return []*models.Experiment{&models.Experiment{ID: "exp_123", Name: "Experiment", Branches: []*models.Branch{}}}, nil
			},
		}, &fakeExperimentBranchStore{
			listFn: func(_ context.Context, ids []string) (map[string][]*models.Branch, error) {
				if len(ids) != 1 || ids[0] != "exp_123" {
					t.Fatalf("unexpected ids: %#v", ids)
				}
					return map[string][]*models.Branch{"exp_123": []*models.Branch{&models.Branch{ID: "branch_123", Key: "control"}}}, nil
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/", "", map[string]string{"appID": "app_123"})

		handler.List(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		handler := NewExperimentHandler(&fakeExperimentStore{
			listFn: func(context.Context, string) ([]*models.Experiment, error) { return nil, errors.New("boom") },
		}, &fakeExperimentBranchStore{})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/", "", map[string]string{"appID": "app_123"})
		handler.List(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})

	t.Run("branch load error", func(t *testing.T) {
		handler := NewExperimentHandler(&fakeExperimentStore{
			listFn: func(context.Context, string) ([]*models.Experiment, error) {
					return []*models.Experiment{&models.Experiment{ID: "exp_123"}}, nil
			},
		}, &fakeExperimentBranchStore{
			listFn: func(context.Context, []string) (map[string][]*models.Branch, error) { return nil, errors.New("boom") },
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/", "", map[string]string{"appID": "app_123"})
		handler.List(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})
}

func TestExperimentHandlerGetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := NewExperimentHandler(&fakeExperimentStore{
			getFn: func(context.Context, string, string) (*models.Experiment, error) {
				return &models.Experiment{ID: "exp_123", Branches: []*models.Branch{}}, nil
			},
		}, &fakeExperimentBranchStore{
			listFn: func(context.Context, []string) (map[string][]*models.Branch, error) {
					return map[string][]*models.Branch{"exp_123": []*models.Branch{&models.Branch{ID: "branch_123"}}}, nil
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/", "", map[string]string{"appID": "app_123", "id": "exp_123"})
		handler.GetByID(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := NewExperimentHandler(&fakeExperimentStore{
			getFn: func(context.Context, string, string) (*models.Experiment, error) { return nil, store.ErrNotFound },
		}, &fakeExperimentBranchStore{})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/", "", map[string]string{"appID": "app_123", "id": "exp_123"})
		handler.GetByID(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("branch load error", func(t *testing.T) {
		handler := NewExperimentHandler(&fakeExperimentStore{
			getFn: func(context.Context, string, string) (*models.Experiment, error) {
				return &models.Experiment{ID: "exp_123", Branches: []*models.Branch{}}, nil
			},
		}, &fakeExperimentBranchStore{
			listFn: func(context.Context, []string) (map[string][]*models.Branch, error) { return nil, errors.New("boom") },
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodGet, "/", "", map[string]string{"appID": "app_123", "id": "exp_123"})
		handler.GetByID(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})
}

func TestExperimentHandlerUpdate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var got store.UpdateExperimentParams
		handler := NewExperimentHandler(&fakeExperimentStore{
			updateFn: func(_ context.Context, appID, id string, p store.UpdateExperimentParams) (*models.Experiment, error) {
				got = p
				return &models.Experiment{ID: id, Name: p.Name, Branches: []*models.Branch{}}, nil
			},
		}, &fakeExperimentBranchStore{
			listFn: func(context.Context, []string) (map[string][]*models.Branch, error) {
					return map[string][]*models.Branch{"exp_123": []*models.Branch{&models.Branch{ID: "branch_123"}}}, nil
			},
		})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodPut, "/", `{"name":"  Updated  ","status":"active"}`, map[string]string{"appID": "app_123", "id": "exp_123"})

		handler.Update(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
		if got.Name != "Updated" || got.Status != models.ExperimentStatusActive {
			t.Fatalf("unexpected params: %#v", got)
		}
	})

	testCases := []struct {
		name       string
		body       string
		err        error
		wantStatus int
	}{
		{"invalid body", "{", nil, http.StatusBadRequest},
		{"missing name", `{"name":" "}`, nil, http.StatusUnprocessableEntity},
		{"invalid status", `{"name":"Exp","status":"archived"}`, nil, http.StatusUnprocessableEntity},
		{"not found", `{"name":"Exp","status":"active"}`, store.ErrNotFound, http.StatusNotFound},
		{"unexpected", `{"name":"Exp","status":"active"}`, errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewExperimentHandler(&fakeExperimentStore{
				updateFn: func(context.Context, string, string, store.UpdateExperimentParams) (*models.Experiment, error) {
					return nil, tc.err
				},
			}, &fakeExperimentBranchStore{})
			rec := httptest.NewRecorder()
			req := newRequestWithURLParams(http.MethodPut, "/", tc.body, map[string]string{"appID": "app_123", "id": "exp_123"})
			handler.Update(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestExperimentHandlerDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := NewExperimentHandler(&fakeExperimentStore{
			deleteFn: func(context.Context, string, string) error { return nil },
		}, &fakeExperimentBranchStore{})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodDelete, "/", "", map[string]string{"appID": "app_123", "id": "exp_123"})
		handler.Delete(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := NewExperimentHandler(&fakeExperimentStore{
			deleteFn: func(context.Context, string, string) error { return store.ErrNotFound },
		}, &fakeExperimentBranchStore{})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodDelete, "/", "", map[string]string{"appID": "app_123", "id": "exp_123"})
		handler.Delete(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("unexpected error", func(t *testing.T) {
		handler := NewExperimentHandler(&fakeExperimentStore{
			deleteFn: func(context.Context, string, string) error { return errors.New("boom") },
		}, &fakeExperimentBranchStore{})
		rec := httptest.NewRecorder()
		req := newRequestWithURLParams(http.MethodDelete, "/", "", map[string]string{"appID": "app_123", "id": "exp_123"})
		handler.Delete(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})
}
